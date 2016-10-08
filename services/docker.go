package services

import (
	"log"
	"strings"

	ptypes "github.com/franela/play-with-docker/types"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

var c *client.Client

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
	opts := types.NetworkCreate{Attachable: true, Driver: "overlay"}
	_, err := c.NetworkCreate(context.Background(), name, opts)

	if err != nil {
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

func CreateExecConnection(id string, ctx context.Context) (string, error) {
	conf := types.ExecConfig{Tty: true, AttachStdin: true, AttachStderr: true, AttachStdout: true, Cmd: []string{"sh"}}
	resp, err := c.ContainerExecCreate(ctx, id, conf)
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

func AttachExecConnection(execId string, ctx context.Context) (*types.HijackedResponse, error) {
	conf := types.ExecConfig{Tty: true, AttachStdin: true, AttachStderr: true, AttachStdout: true, Cmd: []string{"sh"}}
	conn, err := c.ContainerExecAttach(ctx, execId, conf)

	if err != nil {
		return nil, err
	}

	return &conn, nil
}

func CreateInstance(net string) (*ptypes.Instance, error) {

	h := &container.HostConfig{NetworkMode: container.NetworkMode(net), Privileged: true}
	conf := &container.Config{Image: "docker:dind"}
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

	return &ptypes.Instance{Name: strings.Replace(cinfo.Name, "/", "", 1), IP: cinfo.NetworkSettings.Networks[net].IPAddress}, nil
}

func DeleteContainer(id string) error {
	return c.ContainerRemove(context.Background(), id, types.ContainerRemoveOptions{Force: true})
}
