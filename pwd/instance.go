package pwd

import (
	"io"
	"log"
	"net"
	"time"

	"github.com/play-with-docker/play-with-docker/config"
	"github.com/play-with-docker/play-with-docker/event"
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
	instance, err := p.storage.InstanceGet(name)
	if err != nil {
		log.Println(err)
		return nil
	}
	return instance
}

func (p *pwd) InstanceFindBySession(session *types.Session) ([]*types.Instance, error) {
	defer observeAction("InstanceFindBySession", time.Now())
	instances, err := p.storage.InstanceFindBySessionId(session.Id)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return instances, nil
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

	if err := p.storage.InstanceDelete(instance.Name); err != nil {
		return err
	}

	p.event.Emit(event.INSTANCE_DELETE, session.Id, instance.Name)

	p.setGauges()

	return nil
}

func (p *pwd) InstanceNew(session *types.Session, conf types.InstanceConfig) (*types.Instance, error) {
	defer observeAction("InstanceNew", time.Now())

	prov, err := p.getProvisioner(conf.Type)
	if err != nil {
		return nil, err
	}

	if config.ForceTLS {
		conf.Tls = true
	}

	instance, err := prov.InstanceNew(session, conf)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	err = p.storage.InstancePut(instance)
	if err != nil {
		return nil, err
	}

	p.event.Emit(event.INSTANCE_NEW, session.Id, instance.Name, instance.IP, instance.Hostname, instance.ProxyHost)

	p.setGauges()

	return instance, nil
}

func (p *pwd) InstanceExec(instance *types.Instance, cmd []string) (int, error) {
	defer observeAction("InstanceExec", time.Now())

	prov, err := p.getProvisioner(instance.Type)
	if err != nil {
		return -1, err
	}
	exitCode, err := prov.InstanceExec(instance, cmd)
	if err != nil {
		log.Println(err)
		return -1, err
	}
	return exitCode, nil
}

func (p *pwd) InstanceFSTree(instance *types.Instance) (io.Reader, error) {
	defer observeAction("InstanceFSTree", time.Now())

	prov, err := p.getProvisioner(instance.Type)
	if err != nil {
		return nil, err
	}
	return prov.InstanceFSTree(instance)
}

func (p *pwd) InstanceFile(instance *types.Instance, filePath string) (io.Reader, error) {
	defer observeAction("InstanceFile", time.Now())

	prov, err := p.getProvisioner(instance.Type)
	if err != nil {
		return nil, err
	}
	return prov.InstanceFile(instance, filePath)
}
