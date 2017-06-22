package pwd

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/play-with-docker/play-with-docker/config"
	"github.com/play-with-docker/play-with-docker/docker"
	"github.com/play-with-docker/play-with-docker/pwd/types"

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

func (p *pwd) InstanceResizeTerminal(instance *types.Instance, rows, cols uint) error {
	defer observeAction("InstanceResizeTerminal", time.Now())
	return p.docker.ContainerResize(instance.Name, rows, cols)
}

func (p *pwd) InstanceAttachTerminal(instance *types.Instance) error {
	conn, err := p.docker.CreateAttachConnection(instance.Name)

	if err != nil {
		return err
	}

	encoder := encoding.Replacement.NewEncoder()
	sw := &sessionWriter{sessionId: instance.Session.Id, instanceName: instance.Name, broadcast: p.broadcast}
	instance.Terminal = conn
	io.Copy(encoder.Writer(sw), conn)

	return nil
}

func (p *pwd) InstanceUploadFromUrl(instance *types.Instance, url string) error {
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

func (p *pwd) InstanceUploadFromReader(instance *types.Instance, fileName string, reader io.Reader) error {
	defer observeAction("InstanceUploadFromReader", time.Now())

	copyErr := p.docker.CopyToContainer(instance.Name, "/var/run/pwd/uploads", fileName, reader)

	if copyErr != nil {
		return fmt.Errorf("Error while uploading file [%s]. Error: %s\n", fileName, copyErr)
	}

	return nil
}

func (p *pwd) InstanceGet(session *types.Session, name string) *types.Instance {
	defer observeAction("InstanceGet", time.Now())
	return session.Instances[name]
}

func (p *pwd) InstanceFindByIP(ip string) *types.Instance {
	defer observeAction("InstanceFindByIP", time.Now())
	i, err := p.storage.InstanceFindByIP(ip)
	if err != nil {
		return nil
	}

	return i
}

func (p *pwd) InstanceFindByIPAndSession(sessionPrefix, ip string) *types.Instance {
	defer observeAction("InstanceFindByIPAndSession", time.Now())
	i, err := p.storage.InstanceFindByIPAndSession(sessionPrefix, ip)
	if err != nil {
		return nil
	}

	return i
}

func (p *pwd) InstanceFindByAlias(sessionPrefix, alias string) *types.Instance {
	defer observeAction("InstanceFindByAlias", time.Now())
	i, err := p.storage.InstanceFindByAlias(sessionPrefix, alias)
	if err != nil {
		return nil
	}
	return i
}

func (p *pwd) InstanceDelete(session *types.Session, instance *types.Instance) error {
	defer observeAction("InstanceDelete", time.Now())
	if instance.Terminal != nil {
		instance.Terminal.Close()
	}
	err := p.docker.DeleteContainer(instance.Name)
	if err != nil && !strings.Contains(err.Error(), "No such container") {
		log.Println(err)
		return err
	}

	p.broadcast.BroadcastTo(session.Id, "delete instance", instance.Name)

	delete(session.Instances, instance.Name)
	if err := p.storage.SessionPut(session); err != nil {
		return err
	}

	p.setGauges()

	return nil
}

func (p *pwd) checkHostnameExists(session *types.Session, hostname string) bool {
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

func (p *pwd) InstanceNew(session *types.Session, conf InstanceConfig) (*types.Instance, error) {
	defer observeAction("InstanceNew", time.Now())
	session.Lock()
	defer session.Unlock()

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

	instance := &types.Instance{}
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
	instance.Session = session
	// For now this condition holds through. In the future we might need a more complex logic.
	instance.IsDockerHost = opts.Privileged

	if session.Instances == nil {
		session.Instances = make(map[string]*types.Instance)
	}
	session.Instances[instance.Name] = instance

	go p.InstanceAttachTerminal(instance)

	err = p.storage.SessionPut(session)
	if err != nil {
		return nil, err
	}

	p.broadcast.BroadcastTo(session.Id, "new instance", instance.Name, instance.IP, instance.Hostname)

	p.setGauges()

	return instance, nil
}

func (p *pwd) InstanceWriteToTerminal(instance *types.Instance, data string) {
	defer observeAction("InstanceWriteToTerminal", time.Now())
	if instance != nil && instance.Terminal != nil && len(data) > 0 {
		instance.Terminal.Write([]byte(data))
	}
}

func (p *pwd) InstanceAllowedImages() []string {
	defer observeAction("InstanceAllowedImages", time.Now())

	return []string{
		config.GetDindImageName(),
		"franela/dind:overlay2-dev",
		"franela/ucp:2.4.1",
	}

}

func (p *pwd) InstanceExec(instance *types.Instance, cmd []string) (int, error) {
	defer observeAction("InstanceExec", time.Now())
	return p.docker.Exec(instance.Name, cmd)
}
