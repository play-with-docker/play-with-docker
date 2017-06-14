package pwd

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/play-with-docker/play-with-docker/config"
	"github.com/play-with-docker/play-with-docker/docker"

	"golang.org/x/text/encoding"
)

type sessionWriter struct {
	sessionId    string
	instanceName string
	broadcast    BroadcastApi
}

func (s *sessionWriter) Write(p []byte) (n int, err error) {
	s.broadcast.BroadcastTo(s.sessionId, "terminal out", s.instanceName, string(p))
	return len(p), nil
}

type UInt16Slice []uint16

func (p UInt16Slice) Len() int           { return len(p) }
func (p UInt16Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p UInt16Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

type Instance struct {
	Image        string           `json:"image"`
	Name         string           `json:"name"`
	Hostname     string           `json:"hostname"`
	IP           string           `json:"ip"`
	IsManager    *bool            `json:"is_manager"`
	Mem          string           `json:"mem"`
	Cpu          string           `json:"cpu"`
	Alias        string           `json:"alias"`
	ServerCert   []byte           `json:"server_cert"`
	ServerKey    []byte           `json:"server_key"`
	CACert       []byte           `json:"ca_cert"`
	Cert         []byte           `json:"cert"`
	Key          []byte           `json:"key"`
	IsDockerHost bool             `json:"is_docker_host"`
	session      *Session         `json:"-"`
	conn         net.Conn         `json:"-"`
	ctx          context.Context  `json:"-"`
	docker       docker.DockerApi `json:"-"`
	tempPorts    []uint16         `json:"-"`
	Ports        UInt16Slice
	rw           sync.Mutex
}
type InstanceConfig struct {
	ImageName  string
	Alias      string
	Hostname   string
	ServerCert []byte
	ServerKey  []byte
	CACert     []byte
	Cert       []byte
	Key        []byte
	Host       string
}

func (i *Instance) setUsedPort(port uint16) {
	i.rw.Lock()
	defer i.rw.Unlock()

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

func (p *pwd) InstanceResizeTerminal(instance *Instance, rows, cols uint) error {
	defer observeAction("InstanceResizeTerminal", time.Now())
	return p.docker.ContainerResize(instance.Name, rows, cols)
}

func (p *pwd) InstanceAttachTerminal(instance *Instance) error {
	conn, err := p.docker.CreateAttachConnection(instance.Name)

	if err != nil {
		return err
	}

	encoder := encoding.Replacement.NewEncoder()
	sw := &sessionWriter{sessionId: instance.session.Id, instanceName: instance.Name, broadcast: p.broadcast}
	instance.conn = conn
	io.Copy(encoder.Writer(sw), conn)

	return nil
}

func (p *pwd) InstanceUploadFromUrl(instance *Instance, url string) error {
	defer observeAction("InstanceUploadFromUrl", time.Now())
	log.Printf("Downloading file [%s]\n", url)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("Could not download file [%s]. Error: %s\n", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("Could not download file [%s]. Status code: %d\n", url, resp.StatusCode)
	}

	_, fileName := filepath.Split(url)

	copyErr := p.docker.CopyToContainer(instance.Name, "/var/run/pwd/uploads", fileName, resp.Body)

	if copyErr != nil {
		return fmt.Errorf("Error while downloading file [%s]. Error: %s\n", url, copyErr)
	}

	return nil
}

func (p *pwd) InstanceGet(session *Session, name string) *Instance {
	defer observeAction("InstanceGet", time.Now())
	return session.Instances[name]
}

func (p *pwd) InstanceFindByIP(ip string) *Instance {
	defer observeAction("InstanceFindByIP", time.Now())
	for _, s := range sessions {
		for _, i := range s.Instances {
			if i.IP == ip {
				return i
			}
		}
	}
	return nil
}

func (p *pwd) InstanceFindByAlias(sessionPrefix, alias string) *Instance {
	defer observeAction("InstanceFindByAlias", time.Now())
	for id, s := range sessions {
		if strings.HasPrefix(id, sessionPrefix) {
			for _, i := range s.Instances {
				if i.Alias == alias {
					return i
				}
			}
		}
	}
	return nil
}

func (p *pwd) InstanceDelete(session *Session, instance *Instance) error {
	defer observeAction("InstanceDelete", time.Now())
	if instance.conn != nil {
		instance.conn.Close()
	}
	err := p.docker.DeleteContainer(instance.Name)
	if err != nil && !strings.Contains(err.Error(), "No such container") {
		log.Println(err)
		return err
	}

	p.broadcast.BroadcastTo(session.Id, "delete instance", instance.Name)

	delete(session.Instances, instance.Name)
	if err := p.storage.Save(); err != nil {
		return err
	}

	setGauges()

	return nil
}

func (p *pwd) checkHostnameExists(session *Session, hostname string) bool {
	containerName := fmt.Sprintf("%s_%s", session.Id[:8], hostname)
	exists := false
	for _, instance := range session.Instances {
		if instance.Name == containerName {
			exists = true
			break
		}
	}
	return exists
}

func (p *pwd) InstanceNew(session *Session, conf InstanceConfig) (*Instance, error) {
	defer observeAction("InstanceNew", time.Now())
	session.rw.Lock()
	defer session.rw.Unlock()

	if conf.ImageName == "" {
		conf.ImageName = config.GetDindImageName()
	}
	log.Printf("NewInstance - using image: [%s]\n", conf.ImageName)

	if conf.Hostname == "" {
		var nodeName string
		for i := 1; ; i++ {
			nodeName = fmt.Sprintf("node%d", i)
			exists := p.checkHostnameExists(session, nodeName)
			if !exists {
				break
			}
		}
		conf.Hostname = nodeName
	}
	containerName := fmt.Sprintf("%s_%s", session.Id[:8], conf.Hostname)

	opts := docker.CreateContainerOpts{
		Image:         conf.ImageName,
		SessionId:     session.Id,
		PwdIpAddress:  session.PwdIpAddress,
		ContainerName: containerName,
		Hostname:      conf.Hostname,
		ServerCert:    conf.ServerCert,
		ServerKey:     conf.ServerKey,
		CACert:        conf.CACert,
		Privileged:    false,
		HostFQDN:      conf.Host,
	}

	for _, imageName := range p.InstanceAllowedImages() {
		if conf.ImageName == imageName {
			opts.Privileged = true
			break
		}
	}

	ip, err := p.docker.CreateContainer(opts)
	if err != nil {
		return nil, err
	}

	instance := &Instance{}
	instance.Image = opts.Image
	instance.IP = ip
	instance.Name = containerName
	instance.Hostname = conf.Hostname
	instance.Alias = conf.Alias
	instance.Cert = conf.Cert
	instance.Key = conf.Key
	instance.ServerCert = conf.ServerCert
	instance.ServerKey = conf.ServerKey
	instance.CACert = conf.CACert
	instance.session = session
	// For now this condition holds through. In the future we might need a more complex logic.
	instance.IsDockerHost = opts.Privileged

	if session.Instances == nil {
		session.Instances = make(map[string]*Instance)
	}
	session.Instances[instance.Name] = instance

	go p.InstanceAttachTerminal(instance)

	err = p.storage.Save()
	if err != nil {
		return nil, err
	}

	p.broadcast.BroadcastTo(session.Id, "new instance", instance.Name, instance.IP, instance.Hostname)

	setGauges()

	return instance, nil
}

func (p *pwd) InstanceWriteToTerminal(instance *Instance, data string) {
	defer observeAction("InstanceWriteToTerminal", time.Now())
	if instance != nil && instance.conn != nil && len(data) > 0 {
		instance.conn.Write([]byte(data))
	}
}

func (p *pwd) InstanceAllowedImages() []string {
	defer observeAction("InstanceAllowedImages", time.Now())

	return []string{
		config.GetDindImageName(),
		"franela/dind:overlay2-dev",
	}

}

func (p *pwd) InstanceExec(instance *Instance, cmd []string) (int, error) {
	defer observeAction("InstanceExec", time.Now())
	return p.docker.Exec(instance.Name, cmd)
}
