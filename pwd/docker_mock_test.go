package pwd

import (
	"io"
	"net"

	"github.com/docker/docker/api/types"
	"github.com/play-with-docker/play-with-docker/docker"
)

type mockDocker struct {
	createNetwork   func(string) error
	connectNetwork  func(container, network, ip string) (string, error)
	containerResize func(string, uint, uint) error
}

func (m *mockDocker) CreateNetwork(id string) error {
	if m.createNetwork == nil {
		return nil
	}
	return m.createNetwork(id)
}
func (m *mockDocker) ConnectNetwork(container, network, ip string) (string, error) {
	if m.connectNetwork == nil {
		return "10.0.0.1", nil
	}
	return m.connectNetwork(container, network, ip)
}

func (m *mockDocker) GetDaemonInfo() (types.Info, error) {
	return types.Info{}, nil
}

func (m *mockDocker) GetSwarmPorts() ([]string, []uint16, error) {
	return []string{}, []uint16{}, nil
}
func (m *mockDocker) GetPorts() ([]uint16, error) {
	return []uint16{}, nil
}
func (m *mockDocker) GetContainerStats(name string) (io.ReadCloser, error) {
	return nil, nil
}
func (m *mockDocker) ContainerResize(name string, rows, cols uint) error {
	if m.containerResize != nil {
		return m.containerResize(name, rows, cols)
	}
	return nil
}
func (m *mockDocker) CreateAttachConnection(name string) (net.Conn, error) {
	return nil, nil
}
func (m *mockDocker) CopyToContainer(containerName, destination, fileName string, content io.Reader) error {
	return nil
}
func (m *mockDocker) DeleteContainer(id string) error {
	return nil
}
func (m *mockDocker) CreateContainer(opts docker.CreateContainerOpts) (string, error) {
	return "", nil
}
func (m *mockDocker) ExecAttach(instanceName string, command []string, out io.Writer) (int, error) {
	return 0, nil
}
func (m *mockDocker) DisconnectNetwork(containerId, networkId string) error {
	return nil
}
func (m *mockDocker) DeleteNetwork(id string) error {
	return nil
}
func (m *mockDocker) Exec(instanceName string, command []string) (int, error) {
	return 0, nil
}
