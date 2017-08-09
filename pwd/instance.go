package pwd

import (
	"io"
	"log"
	"net"
	"time"

	"github.com/play-with-docker/play-with-docker/event"
	"github.com/play-with-docker/play-with-docker/provisioner"
	"github.com/play-with-docker/play-with-docker/pwd/types"
)

func (p *pwd) InstanceResizeTerminal(instance *types.Instance, rows, cols uint) error {
	defer observeAction("InstanceResizeTerminal", time.Now())
	prov, err := p.getProvisioner(instance.Type)
	if err != nil {
		return err
	}
	return prov.InstanceResizeTerminal(instance, rows, cols)
}

func (p *pwd) InstanceGetTerminal(instance *types.Instance) (net.Conn, error) {
	defer observeAction("InstanceGetTerminal", time.Now())
	prov, err := p.getProvisioner(instance.Type)
	if err != nil {
		return nil, err
	}
	return prov.InstanceGetTerminal(instance)
}

func (p *pwd) InstanceUploadFromUrl(instance *types.Instance, fileName, dest string, url string) error {
	defer observeAction("InstanceUploadFromUrl", time.Now())
	prov, err := p.getProvisioner(instance.Type)
	if err != nil {
		return err
	}

	return prov.InstanceUploadFromUrl(instance, fileName, dest, url)
}

func (p *pwd) InstanceUploadFromReader(instance *types.Instance, fileName, dest string, reader io.Reader) error {
	defer observeAction("InstanceUploadFromReader", time.Now())

	prov, err := p.getProvisioner(instance.Type)
	if err != nil {
		return err
	}

	return prov.InstanceUploadFromReader(instance, fileName, dest, reader)
}

func (p *pwd) InstanceGet(session *types.Session, name string) *types.Instance {
	defer observeAction("InstanceGet", time.Now())
	instance, err := p.storage.InstanceGet(session.Id, name)
	if err != nil {
		log.Println(err)
		return nil
	}
	return instance
}

func (p *pwd) InstanceFindByIP(sessionId, ip string) *types.Instance {
	defer observeAction("InstanceFindByIP", time.Now())
	i, err := p.storage.InstanceFindByIP(sessionId, ip)
	if err != nil {
		return nil
	}

	return i
}

func (p *pwd) InstanceDelete(session *types.Session, instance *types.Instance) error {
	defer observeAction("InstanceDelete", time.Now())

	prov, err := p.getProvisioner(instance.Type)
	if err != nil {
		return err
	}

	err = prov.InstanceDelete(session, instance)
	if err != nil {
		log.Println(err)
		return err
	}

	p.event.Emit(event.INSTANCE_DELETE, session.Id, instance.Name)

	if err := p.storage.InstanceDelete(session.Id, instance.Name); err != nil {
		return err
	}

	p.setGauges()

	return nil
}

func (p *pwd) InstanceNew(session *types.Session, conf types.InstanceConfig) (*types.Instance, error) {
	defer observeAction("InstanceNew", time.Now())
	session.Lock()
	defer session.Unlock()

	prov, err := p.getProvisioner(conf.Type)
	if err != nil {
		return nil, err
	}
	instance, err := prov.InstanceNew(session, conf)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	if session.Instances == nil {
		session.Instances = make(map[string]*types.Instance)
	}
	session.Instances[instance.Name] = instance

	err = p.storage.InstanceCreate(session.Id, instance)
	if err != nil {
		return nil, err
	}

	p.event.Emit(event.INSTANCE_NEW, session.Id, instance.Name, instance.IP, instance.Hostname)

	p.setGauges()

	return instance, nil
}

func (p *pwd) InstanceExec(instance *types.Instance, cmd []string) (int, error) {
	defer observeAction("InstanceExec", time.Now())
	return p.docker(instance.SessionId).Exec(instance.Name, cmd)
}

func (p *pwd) InstanceAllowedImages() []string {
	defer observeAction("InstanceAllowedImages", time.Now())

	return p.dindProvisioner.(*provisioner.DinD).InstanceAllowedImages()
}
