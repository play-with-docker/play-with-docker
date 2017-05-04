package services

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

var c *client.Client

const (
	Byte     = 1
	Kilobyte = 1024 * Byte
	Megabyte = 1024 * Kilobyte
)

func init() {
	var err error
	c, err = client.NewEnvClient()
	if err != nil {
		// this wont happen if daemon is offline, only for some critical errors
		log.Fatal("Cannot initialize docker client")
	}

}

func GetContainerStats(id string) (io.ReadCloser, error) {
	stats, err := c.ContainerStats(context.Background(), id, false)

	return stats.Body, err
}

func GetContainerInfo(id string) (types.ContainerJSON, error) {
	return c.ContainerInspect(context.Background(), id)
}

func GetDaemonInfo(i *Instance) (types.Info, error) {
	if i.dockerClient == nil {
		return types.Info{}, fmt.Errorf("Docker client for DinD (%s) is not ready", i.IP)
	}
	return i.dockerClient.Info(context.Background())
}

func SetInstanceSwarmPorts(i *Instance) error {
	if i.dockerClient == nil {
		return fmt.Errorf("Docker client for DinD (%s) is not ready", i.IP)
	}

	hostnamesIdx := map[string]*Instance{}
	for _, ins := range i.session.Instances {
		hostnamesIdx[ins.Hostname] = ins
	}

	nodesIdx := map[string]*Instance{}
	nodes, nodesErr := i.dockerClient.NodeList(context.Background(), types.NodeListOptions{})
	if nodesErr != nil {
		return nodesErr
	}
	for _, n := range nodes {
		nodesIdx[n.ID] = hostnamesIdx[n.Description.Hostname]
	}

	tasks, err := i.dockerClient.TaskList(context.Background(), types.TaskListOptions{})
	if err != nil {
		return err
	}
	services := map[string][]uint16{}
	for _, t := range tasks {
		services[t.ServiceID] = []uint16{}
	}
	for serviceID, _ := range services {
		s, _, err := i.dockerClient.ServiceInspectWithRaw(context.Background(), serviceID, types.ServiceInspectOptions{})
		if err != nil {
			return err
		}
		for _, p := range s.Endpoint.Ports {
			services[serviceID] = append(services[serviceID], uint16(p.PublishedPort))
		}
	}
	for _, t := range tasks {
		for _, n := range nodes {
			ins := nodesIdx[n.ID]
			if ins != nil {
				for _, p := range services[t.ServiceID] {
					ins.setUsedPort(p)
				}
			}
		}
	}

	return nil
}

func GetUsedPorts(i *Instance) ([]uint16, error) {
	if i.dockerClient == nil {
		return nil, fmt.Errorf("Docker client for DinD (%s) is not ready", i.IP)
	}
	opts := types.ContainerListOptions{}
	containers, err := i.dockerClient.ContainerList(context.Background(), opts)
	if err != nil {
		return nil, err
	}

	openPorts := []uint16{}
	for _, c := range containers {
		for _, p := range c.Ports {
			// When port is not published on the host docker return public port as 0, so we need to avoid it
			if p.PublicPort != 0 {
				openPorts = append(openPorts, p.PublicPort)
			}
		}
	}

	return openPorts, nil
}

func CreateNetwork(name string) error {
	opts := types.NetworkCreate{Driver: "overlay", Attachable: true}
	_, err := c.NetworkCreate(context.Background(), name, opts)

	if err != nil {
		log.Printf("Starting session err [%s]\n", err)

		return err
	}

	return nil
}
func ConnectNetwork(containerId, networkId, ip string) (string, error) {
	settings := &network.EndpointSettings{}
	if ip != "" {
		settings.IPAddress = ip
	}
	err := c.NetworkConnect(context.Background(), networkId, containerId, settings)

	if err != nil && !strings.Contains(err.Error(), "already exists") {
		log.Printf("Connection container to network err [%s]\n", err)

		return "", err
	}

	// Obtain the IP of the PWD container in this network
	container, err := c.ContainerInspect(context.Background(), containerId)
	if err != nil {
		return "", err
	}

	n, found := container.NetworkSettings.Networks[networkId]
	if !found {
		return "", fmt.Errorf("Container [%s] connected to the network [%s] but couldn't obtain it's IP address", containerId, networkId)
	}

	return n.IPAddress, nil
}

func DisconnectNetwork(containerId, networkId string) error {
	err := c.NetworkDisconnect(context.Background(), networkId, containerId, true)

	if err != nil {
		log.Printf("Disconnection of container from network err [%s]\n", err)

		return err
	}

	return nil
}

func DeleteNetwork(id string) error {
	err := c.NetworkRemove(context.Background(), id)

	if err != nil {
		return err
	}

	return nil
}

func CreateAttachConnection(id string, ctx context.Context) (*types.HijackedResponse, error) {

	conf := types.ContainerAttachOptions{true, true, true, true, "ctrl-^,ctrl-^", true}
	conn, err := c.ContainerAttach(ctx, id, conf)
	if err != nil {
		return nil, err
	}

	return &conn, nil
}

func ResizeConnection(name string, cols, rows uint) error {
	return c.ContainerResize(context.Background(), name, types.ResizeOptions{Height: rows, Width: cols})
}

func CreateInstance(session *Session, dindImage string) (*Instance, error) {
	h := &container.HostConfig{
		NetworkMode: container.NetworkMode(session.Id),
		Privileged:  true,
		AutoRemove:  true,
		LogConfig:   container.LogConfig{Config: map[string]string{"max-size": "10m", "max-file": "1"}},
	}

	if os.Getenv("APPARMOR_PROFILE") != "" {
		h.SecurityOpt = []string{fmt.Sprintf("apparmor=%s", os.Getenv("APPARMOR_PROFILE"))}
	}

	var pidsLimit = int64(1000)
	if envLimit := os.Getenv("MAX_PROCESSES"); envLimit != "" {
		if i, err := strconv.Atoi(envLimit); err == nil {
			pidsLimit = int64(i)
		}
	}
	h.Resources.PidsLimit = pidsLimit
	h.Resources.Memory = 4092 * Megabyte
	t := true
	h.Resources.OomKillDisable = &t

	var nodeName string
	var containerName string
	for i := 1; ; i++ {
		nodeName = fmt.Sprintf("node%d", i)
		containerName = fmt.Sprintf("%s_%s", session.Id[:8], nodeName)
		exists := false
		for _, instance := range session.Instances {
			if instance.Name == containerName {
				exists = true
				break
			}
		}
		if !exists {
			break
		}
	}
	conf := &container.Config{Hostname: nodeName,
		Image:        dindImage,
		Tty:          true,
		OpenStdin:    true,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Env:          []string{fmt.Sprintf("PWD_IP_ADDRESS=%s", session.PwdIpAddress)},
	}
	networkConf := &network.NetworkingConfig{
		map[string]*network.EndpointSettings{
			session.Id: &network.EndpointSettings{Aliases: []string{nodeName}},
		},
	}
	container, err := c.ContainerCreate(context.Background(), conf, h, networkConf, containerName)

	if err != nil {
		return nil, err
	}

	err = c.ContainerStart(context.Background(), container.ID, types.ContainerStartOptions{})
	if err != nil {
		return nil, err
	}

	cinfo, err := GetContainerInfo(container.ID)
	if err != nil {
		return nil, err
	}

	return &Instance{Name: containerName, Hostname: cinfo.Config.Hostname, IP: cinfo.NetworkSettings.Networks[session.Id].IPAddress}, nil
}

func DeleteContainer(id string) error {
	return c.ContainerRemove(context.Background(), id, types.ContainerRemoveOptions{Force: true, RemoveVolumes: true})
}
