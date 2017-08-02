package provisioner

import (
	"io"
	"net"

	"github.com/play-with-docker/play-with-docker/pwd/types"
)

type ProvisionerApi interface {
	InstanceNew(session *types.Session, conf types.InstanceConfig) (*types.Instance, error)
	InstanceDelete(session *types.Session, instance *types.Instance) error

	InstanceResizeTerminal(instance *types.Instance, cols, rows uint) error
	InstanceGetTerminal(instance *types.Instance) (net.Conn, error)

	InstanceUploadFromUrl(instance *types.Instance, fileName, dest, url string) error
	InstanceUploadFromReader(instance *types.Instance, fileName, dest string, reader io.Reader) error
}
