package services

import (
	"context"
	"io"
	"log"
	"os"
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
	statsReader  io.ReadCloser           `json:"-"`
	dockerClient *client.Client          `json:"-"`
	IsManager    *bool                   `json:"is_manager"`
	Mem          string                  `json:"mem"`
	Cpu          string                  `json:"cpu"`
	Ports        []uint16                `json:"ports"`
	tempPorts    []uint16                `json:"-"`
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

func NewInstance(session *Session) (*Instance, error) {
	log.Printf("NewInstance - using image: [%s]\n", dindImage)
	instance, err := CreateInstance(session, dindImage)
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
	//TODO: Use redis
	return session.Instances[name]
}
func DeleteInstance(session *Session, instance *Instance) error {
	// stop collecting stats
	if instance.statsReader != nil {
		instance.statsReader.Close()
	}

	//TODO: Use redis
	delete(session.Instances, instance.Name)
	err := DeleteContainer(instance.Name)

	wsServer.BroadcastTo(session.Id, "delete instance", instance.Name)

	return err
}
