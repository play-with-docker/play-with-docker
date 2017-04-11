package services

import (
	"context"
	"crypto/tls"
	"io"
	"log"
	"os"
	"strings"
	"sync"

	"golang.org/x/text/encoding"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

var rw sync.Mutex

type Instance struct {
	session      *Session                `json:"-"`
	Name         string                  `json:"name"`
	Hostname     string                  `json:"hostname"`
	IP           string                  `json:"ip"`
	conn         *types.HijackedResponse `json:"-"`
	ctx          context.Context         `json:"-"`
	dockerClient *client.Client          `json:"-"`
	IsManager    *bool                   `json:"is_manager"`
	Mem          string                  `json:"mem"`
	Cpu          string                  `json:"cpu"`
	Ports        []uint16                `json:"ports"`
	tempPorts    []uint16                `json:"-"`
	ServerCert   []byte                  `json:"server_cert"`
	ServerKey    []byte                  `json:"server_key"`
	cert         *tls.Certificate        `json:"-"`
}

func (i *Instance) setUsedPort(port uint16) {
	rw.Lock()
	defer rw.Unlock()

	for _, p := range i.tempPorts {
		if p == port {
			return
		}
	}
	i.tempPorts = append(i.tempPorts, port)
}

func (i *Instance) SetCertificate(cert, key []byte) (*tls.Certificate, error) {
	i.ServerCert = cert
	i.ServerKey = key
	c, e := tls.X509KeyPair(i.ServerCert, i.ServerKey)
	if e != nil {
		return nil, e
	}
	i.cert = &c

	// We store sessions as soon as we set instance keys
	if err := saveSessionsToDisk(); err != nil {
		return nil, err
	}
	return i.cert, nil
}
func (i *Instance) GetCertificate() *tls.Certificate {
	return i.cert
}

func (i *Instance) IsConnected() bool {
	return i.conn != nil

}

func (i *Instance) SetSession(s *Session) {
	i.session = s
}

var dindImage string
var defaultDindImageName string

func init() {
	dindImage = getDindImageName()
}

func getDindImageName() string {
	dindImage := os.Getenv("DIND_IMAGE")
	defaultDindImageName = "franela/dind"
	if len(dindImage) == 0 {
		dindImage = defaultDindImageName
	}
	return dindImage
}

func NewInstance(session *Session, imageName string) (*Instance, error) {
	if imageName == "" {
		imageName = dindImage
	}
	log.Printf("NewInstance - using image: [%s]\n", imageName)
	instance, err := CreateInstance(session, imageName)
	if err != nil {
		return nil, err
	}
	instance.session = session

	if session.Instances == nil {
		session.Instances = make(map[string]*Instance)
	}
	session.Instances[instance.Name] = instance

	go instance.Attach()

	err = saveSessionsToDisk()
	if err != nil {
		return nil, err
	}

	wsServer.BroadcastTo(session.Id, "new instance", instance.Name, instance.IP, instance.Hostname)

	setGauges()

	return instance, nil
}

type sessionWriter struct {
	instance *Instance
}

func (s *sessionWriter) Write(p []byte) (n int, err error) {
	wsServer.BroadcastTo(s.instance.session.Id, "terminal out", s.instance.Name, string(p))
	return len(p), nil
}

func (i *Instance) ResizeTerminal(cols, rows uint) error {
	return ResizeConnection(i.Name, cols, rows)
}

func (i *Instance) Attach() {
	i.ctx = context.Background()
	conn, err := CreateAttachConnection(i.Name, i.ctx)

	if err != nil {
		return
	}

	i.conn = conn

	go func() {
		encoder := encoding.Replacement.NewEncoder()
		sw := &sessionWriter{instance: i}
		io.Copy(encoder.Writer(sw), conn.Reader)
	}()

	select {
	case <-i.ctx.Done():
	}
}
func GetInstance(session *Session, name string) *Instance {
	return session.Instances[name]
}

func FindInstanceByIP(ip string) *Instance {
	for _, s := range sessions {
		for _, i := range s.Instances {
			if i.IP == ip {
				return i
			}
		}
	}
	return nil
}

func DeleteInstance(session *Session, instance *Instance) error {
	if instance.conn != nil {
		instance.conn.Close()
	}
	err := DeleteContainer(instance.Name)
	if err != nil && !strings.Contains(err.Error(), "No such container") {
		log.Println(err)
		return err
	}

	wsServer.BroadcastTo(session.Id, "delete instance", instance.Name)

	delete(session.Instances, instance.Name)
	if err := saveSessionsToDisk(); err != nil {
		return err
	}
	setGauges()

	return nil
}
