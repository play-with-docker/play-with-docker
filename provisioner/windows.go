package provisioner

import (
	"fmt"
	"io"
	"net"

	"github.com/play-with-docker/play-with-docker/docker"
	"github.com/play-with-docker/play-with-docker/pwd/types"
)

type windows struct {
	factory docker.FactoryApi
}

func NewWindows(f docker.FactoryApi) *windows {
	return &windows{factory: f}
}

func (d *windows) InstanceNew(session *types.Session, conf types.InstanceConfig) (*types.Instance, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (d *windows) InstanceDelete(session *types.Session, instance *types.Instance) error {
	return fmt.Errorf("Not implemented")
}

func (d *windows) InstanceResizeTerminal(instance *types.Instance, cols, rows uint) error {
	return fmt.Errorf("Not implemented")
}

func (d *windows) InstanceGetTerminal(instance *types.Instance) (net.Conn, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (d *windows) InstanceUploadFromUrl(instance *types.Instance, fileName, dest, url string) error {
	return fmt.Errorf("Not implemented")
}

func (d *windows) InstanceUploadFromReader(instance *types.Instance, fileName, dest string, reader io.Reader) error {
	return fmt.Errorf("Not implemented")
}
