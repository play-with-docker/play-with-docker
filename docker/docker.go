package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/go-connections/tlsconfig"
)

const (
	Byte     = 1
	Kilobyte = 1024 * Byte
	Megabyte = 1024 * Kilobyte
)

type DockerApi interface {
	CreateNetwork(id string) error
	ConnectNetwork(container, network, ip string) (string, error)
	GetDaemonInfo() (types.Info, error)
	GetSwarmPorts() ([]string, []uint16, error)
	GetPorts() ([]uint16, error)
	GetContainerStats(name string) (io.ReadCloser, error)
	ContainerResize(name string, rows, cols uint) error
	CreateAttachConnection(name string) (net.Conn, error)
	CopyToContainer(containerName, destination, fileName string, content io.Reader) error
	DeleteContainer(id string) error
	CreateContainer(opts CreateContainerOpts) (string, error)
	ExecAttach(instanceName string, command []string, out io.Writer) (int, error)
	DisconnectNetwork(containerId, networkId string) error
	DeleteNetwork(id string) error
	Exec(instanceName string, command []string) (int, error)
	New(ip string, cert, key []byte) (DockerApi, error)
	SwarmInit() (*SwarmTokens, error)
	SwarmJoin(addr, token string) error
}

type SwarmTokens struct {
	Manager string
	Worker  string
}

type docker struct {
	c *client.Client
}

func (d *docker) CreateNetwork(id string) error {
	opts := types.NetworkCreate{Driver: "overlay", Attachable: true}
	_, err := d.c.NetworkCreate(context.Background(), id, opts)

	if err != nil {
		log.Printf("Starting session err [%s]\n", err)

		return err
	}

	return nil
}

func (d *docker) ConnectNetwork(containerId, networkId, ip string) (string, error) {
	settings := &network.EndpointSettings{}
	if ip != "" {
		settings.IPAddress = ip
	}
	err := d.c.NetworkConnect(context.Background(), networkId, containerId, settings)

	if err != nil && !strings.Contains(err.Error(), "already exists") {
		log.Printf("Connection container to network err [%s]\n", err)

		return "", err
	}

	// Obtain the IP of the PWD container in this network
	container, err := d.c.ContainerInspect(context.Background(), containerId)
	if err != nil {
		return "", err
	}

	n, found := container.NetworkSettings.Networks[networkId]
	if !found {
		return "", fmt.Errorf("Container [%s] connected to the network [%s] but couldn't obtain it's IP address", containerId, networkId)
	}

	return n.IPAddress, nil
}

func (d *docker) GetDaemonInfo() (types.Info, error) {
	return d.c.Info(context.Background())
}

func (d *docker) GetSwarmPorts() ([]string, []uint16, error) {
	hosts := []string{}
	ports := []uint16{}

	nodesIdx := map[string]string{}
	nodes, nodesErr := d.c.NodeList(context.Background(), types.NodeListOptions{})
	if nodesErr != nil {
		return nil, nil, nodesErr
	}
	for _, n := range nodes {
		nodesIdx[n.ID] = n.Description.Hostname
		hosts = append(hosts, n.Description.Hostname)
	}

	services, err := d.c.ServiceList(context.Background(), types.ServiceListOptions{})
	if err != nil {
		return nil, nil, err
	}
	for _, service := range services {
		for _, p := range service.Endpoint.Ports {
			ports = append(ports, uint16(p.PublishedPort))
		}
	}

	return hosts, ports, nil
}

func (d *docker) GetPorts() ([]uint16, error) {
	opts := types.ContainerListOptions{}
	containers, err := d.c.ContainerList(context.Background(), opts)
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

func (d *docker) GetContainerStats(name string) (io.ReadCloser, error) {
	stats, err := d.c.ContainerStats(context.Background(), name, false)

	return stats.Body, err
}

func (d *docker) ContainerResize(name string, rows, cols uint) error {
	return d.c.ContainerResize(context.Background(), name, types.ResizeOptions{Height: rows, Width: cols})
}

func (d *docker) CreateAttachConnection(name string) (net.Conn, error) {
	ctx := context.Background()

	conf := types.ContainerAttachOptions{true, true, true, true, "ctrl-^,ctrl-^", true}
	conn, err := d.c.ContainerAttach(ctx, name, conf)
	if err != nil {
		return nil, err
	}

	return conn.Conn, nil
}

func (d *docker) CopyToContainer(containerName, destination, fileName string, content io.Reader) error {
	r, w := io.Pipe()
	b, readErr := ioutil.ReadAll(content)
	if readErr != nil {
		return readErr
	}
	t := tar.NewWriter(w)
	go func() {
		t.WriteHeader(&tar.Header{Name: fileName, Mode: 0600, Size: int64(len(b))})
		t.Write(b)
		t.Close()
		w.Close()
	}()
	return d.c.CopyToContainer(context.Background(), containerName, destination, r, types.CopyToContainerOptions{AllowOverwriteDirWithFile: true})
}

func (d *docker) DeleteContainer(id string) error {
	return d.c.ContainerRemove(context.Background(), id, types.ContainerRemoveOptions{Force: true, RemoveVolumes: true})
}

type CreateContainerOpts struct {
	Image         string
	SessionId     string
	PwdIpAddress  string
	ContainerName string
	Hostname      string
	ServerCert    []byte
	ServerKey     []byte
	CACert        []byte
	Privileged    bool
	HostFQDN      string
}

func (d *docker) CreateContainer(opts CreateContainerOpts) (string, error) {
	// Make sure directories are available for the new instance container
	containerDir := "/var/run/pwd"
	containerCertDir := fmt.Sprintf("%s/certs", containerDir)

	env := []string{}

	// Write certs to container cert dir
	if len(opts.ServerCert) > 0 {
		env = append(env, `DOCKER_TLSCERT=\/var\/run\/pwd\/certs\/cert.pem`)
	}
	if len(opts.ServerKey) > 0 {
		env = append(env, `DOCKER_TLSKEY=\/var\/run\/pwd\/certs\/key.pem`)
	}
	if len(opts.CACert) > 0 {
		// if ca cert is specified, verify that clients that connects present a certificate signed by the CA
		env = append(env, `DOCKER_TLSCACERT=\/var\/run\/pwd\/certs\/ca.pem`)
	}
	if len(opts.ServerCert) > 0 || len(opts.ServerKey) > 0 || len(opts.CACert) > 0 {
		// if any of the certs is specified, enable TLS
		env = append(env, "DOCKER_TLSENABLE=true")
	} else {
		env = append(env, "DOCKER_TLSENABLE=false")
	}

	h := &container.HostConfig{
		NetworkMode: container.NetworkMode(opts.SessionId),
		Privileged:  opts.Privileged,
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

	env = append(env, fmt.Sprintf("PWD_IP_ADDRESS=%s", opts.PwdIpAddress))
	env = append(env, fmt.Sprintf("PWD_HOST_FQDN=%s", opts.HostFQDN))
	cf := &container.Config{Hostname: opts.Hostname,
		Image:        opts.Image,
		Tty:          true,
		OpenStdin:    true,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Env:          env,
	}
	networkConf := &network.NetworkingConfig{
		map[string]*network.EndpointSettings{
			opts.SessionId: &network.EndpointSettings{Aliases: []string{opts.Hostname}},
		},
	}
	container, err := d.c.ContainerCreate(context.Background(), cf, h, networkConf, opts.ContainerName)

	if err != nil {
		if client.IsErrImageNotFound(err) {
			log.Printf("Unable to find image '%s' locally\n", opts.Image)
			if err = d.pullImage(context.Background(), opts.Image); err != nil {
				return "", err
			}
			container, err = d.c.ContainerCreate(context.Background(), cf, h, networkConf, opts.ContainerName)
			if err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	}

	if err := d.copyIfSet(opts.ServerCert, "cert.pem", containerCertDir, opts.ContainerName); err != nil {
		return "", err
	}
	if err := d.copyIfSet(opts.ServerKey, "key.pem", containerCertDir, opts.ContainerName); err != nil {
		return "", err
	}
	if err := d.copyIfSet(opts.CACert, "ca.pem", containerCertDir, opts.ContainerName); err != nil {
		return "", err
	}

	err = d.c.ContainerStart(context.Background(), container.ID, types.ContainerStartOptions{})
	if err != nil {
		return "", err
	}

	cinfo, err := d.c.ContainerInspect(context.Background(), container.ID)
	if err != nil {
		return "", err
	}

	return cinfo.NetworkSettings.Networks[opts.SessionId].IPAddress, nil
}

func (d *docker) pullImage(ctx context.Context, image string) error {
	_, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return err
	}

	options := types.ImageCreateOptions{}

	responseBody, err := d.c.ImageCreate(ctx, image, options)
	if err != nil {
		return err
	}
	defer responseBody.Close()

	return jsonmessage.DisplayJSONMessagesStream(
		responseBody,
		os.Stderr,
		os.Stdout.Fd(),
		false,
		nil)
}

func (d *docker) copyIfSet(content []byte, fileName, path, containerName string) error {
	if len(content) > 0 {
		return d.CopyToContainer(containerName, path, fileName, bytes.NewReader(content))
	}
	return nil
}

func (d *docker) ExecAttach(instanceName string, command []string, out io.Writer) (int, error) {
	e, err := d.c.ContainerExecCreate(context.Background(), instanceName, types.ExecConfig{Cmd: command, AttachStdout: true, AttachStderr: true, Tty: true})
	if err != nil {
		return 0, err
	}
	resp, err := d.c.ContainerExecAttach(context.Background(), e.ID, types.ExecConfig{AttachStdout: true, AttachStderr: true, Tty: true})
	if err != nil {
		return 0, err
	}
	io.Copy(out, resp.Reader)
	var ins types.ContainerExecInspect
	for _ = range time.Tick(1 * time.Second) {
		ins, err = d.c.ContainerExecInspect(context.Background(), e.ID)
		if ins.Running {
			continue
		}
		if err != nil {
			return 0, err
		}
		break
	}
	return ins.ExitCode, nil

}

func (d *docker) Exec(instanceName string, command []string) (int, error) {
	e, err := d.c.ContainerExecCreate(context.Background(), instanceName, types.ExecConfig{Cmd: command})
	if err != nil {
		return 0, err
	}
	err = d.c.ContainerExecStart(context.Background(), e.ID, types.ExecStartCheck{})
	if err != nil {
		return 0, err
	}
	var ins types.ContainerExecInspect
	for _ = range time.Tick(1 * time.Second) {
		ins, err = d.c.ContainerExecInspect(context.Background(), e.ID)
		if ins.Running {
			continue
		}
		if err != nil {
			return 0, err
		}
		break
	}
	return ins.ExitCode, nil
}

func (d *docker) DisconnectNetwork(containerId, networkId string) error {
	err := d.c.NetworkDisconnect(context.Background(), networkId, containerId, true)

	if err != nil {
		log.Printf("Disconnection of container from network err [%s]\n", err)

		return err
	}

	return nil
}

func (d *docker) DeleteNetwork(id string) error {
	err := d.c.NetworkRemove(context.Background(), id)

	if err != nil {
		return err
	}

	return nil
}

func (d *docker) New(ip string, cert, key []byte) (DockerApi, error) {
	// We check if the client needs to use TLS
	var tlsConfig *tls.Config
	if len(cert) > 0 && len(key) > 0 {
		tlsConfig = tlsconfig.ClientDefault()
		tlsConfig.InsecureSkipVerify = true
		tlsCert, err := tls.X509KeyPair(cert, key)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{tlsCert}
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   1 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext}
	if tlsConfig != nil {
		transport.TLSClientConfig = tlsConfig
	}
	cli := &http.Client{
		Transport: transport,
	}
	c, err := client.NewClient(fmt.Sprintf("http://%s:2375", ip), api.DefaultVersion, cli, nil)
	if err != nil {
		return nil, fmt.Errorf("Could not connect to DinD docker daemon. %s", err)
	}
	// try to connect up to 5 times and then give up
	for i := 0; i < 5; i++ {
		_, err := c.Ping(context.Background())
		if err != nil {
			if client.IsErrConnectionFailed(err) {
				// connection has failed, maybe instance is not ready yet, sleep and retry
				log.Printf("Connection to [%s] has failed, maybe instance is not ready yet, sleeping and retrying in 1 second. Try #%d\n", fmt.Sprintf("http://%s:2375", ip), i+1)
				time.Sleep(time.Second)
				continue
			}
			return nil, err
		}
	}
	return NewDocker(c), nil
}

func (d *docker) SwarmInit() (*SwarmTokens, error) {
	req := swarm.InitRequest{AdvertiseAddr: "eth0", ListenAddr: "0.0.0.0:2377"}
	_, err := d.c.SwarmInit(context.Background(), req)

	if err != nil {
		return nil, err
	}

	swarmInfo, err := d.c.SwarmInspect(context.Background())
	if err != nil {
		return nil, err
	}

	return &SwarmTokens{
		Worker:  swarmInfo.JoinTokens.Worker,
		Manager: swarmInfo.JoinTokens.Manager,
	}, nil
}
func (d *docker) SwarmJoin(addr, token string) error {
	req := swarm.JoinRequest{RemoteAddrs: []string{addr}, JoinToken: token, ListenAddr: "0.0.0.0:2377", AdvertiseAddr: "eth0"}
	return d.c.SwarmJoin(context.Background(), req)
}

func NewDocker(c *client.Client) *docker {
	return &docker{c: c}
}
