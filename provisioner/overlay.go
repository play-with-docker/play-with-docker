package provisioner

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	dtypes "github.com/docker/docker/api/types"
	"github.com/play-with-docker/play-with-docker/config"
	"github.com/play-with-docker/play-with-docker/docker"
	"github.com/play-with-docker/play-with-docker/pwd/types"
)

type overlaySessionProvisioner struct {
	dockerFactory docker.FactoryApi
}

func NewOverlaySessionProvisioner(df docker.FactoryApi) SessionProvisionerApi {
	return &overlaySessionProvisioner{dockerFactory: df}
}

func (p *overlaySessionProvisioner) SessionNew(s *types.Session) error {
	dockerClient, err := p.dockerFactory.GetForSession(s.Id)
	if err != nil {
		// We assume we are out of capacity
		return fmt.Errorf("Out of capacity")
	}
	u, _ := url.Parse(dockerClient.GetDaemonHost())
	if u.Host == "" {
		s.Host = "localhost"
	} else {
		chunks := strings.Split(u.Host, ":")
		s.Host = chunks[0]
	}

	opts := dtypes.NetworkCreate{Driver: "overlay", Attachable: true}
	if err := dockerClient.CreateNetwork(s.Id, opts); err != nil {
		log.Println("ERROR NETWORKING", err)
		return err
	}
	log.Printf("Network [%s] created for session [%s]\n", s.Id, s.Id)

	ip, err := dockerClient.ConnectNetwork(config.L2ContainerName, s.Id, s.PwdIpAddress)
	if err != nil {
		log.Println(err)
		return err
	}
	s.PwdIpAddress = ip
	log.Printf("Connected %s to network [%s]\n", config.PWDContainerName, s.Id)
	return nil
}
