package services

import (
	"log"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
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

func GetContainerInfo(id string) (types.ContainerJSON, error) {
	return c.ContainerInspect(context.Background(), id)
}

func CreateNetwork(name string) error {
	dockerVersion := os.Getenv("DOCKER_VERSION")
	var opts types.NetworkCreate

	if len(dockerVersion) > 0 && dockerVersion == "1.12" {
		opts = types.NetworkCreate{Attachable: true}
	} else {
		opts = types.NetworkCreate{Attachable: true, Driver: "overlay"}
	}

	_, err := c.NetworkCreate(context.Background(), name, opts)

	if err != nil {
		log.Printf("Starting session err [%s]\n", err)

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

	conf := types.ContainerAttachOptions{true, true, true, true, "ctrl-x,ctrl-x"}
	conn, err := c.ContainerAttach(ctx, id, conf)
	if err != nil {
		return nil, err
	}

	return &conn, nil
}

func ResizeConnection(name string, cols, rows uint) error {
	return c.ContainerResize(context.Background(), name, types.ResizeOptions{Height: rows, Width: cols})
}

func CreateInstance(net string, dindImage string) (*Instance, error) {

	h := &container.HostConfig{NetworkMode: container.NetworkMode(net), Privileged: true}
	h.Resources.PidsLimit = int64(500)
	h.Resources.Memory = 4092 * Megabyte
	t := true
	h.Resources.OomKillDisable = &t

	conf := &container.Config{Image: dindImage, Tty: true, OpenStdin: true, AttachStdin: true, AttachStdout: true, AttachStderr: true}
	container, err := c.ContainerCreate(context.Background(), conf, h, nil, "")

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

	return &Instance{Name: strings.Replace(cinfo.Name, "/", "", 1), Hostname: cinfo.Config.Hostname, IP: cinfo.NetworkSettings.Networks[net].IPAddress}, nil
}

func DeleteContainer(id string) error {
	return c.ContainerRemove(context.Background(), id, types.ContainerRemoveOptions{Force: true, RemoveVolumes: true})
}
