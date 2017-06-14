package types

import (
	"context"
	"net"
	"sync"

	"github.com/play-with-docker/play-with-docker/docker"
)

type UInt16Slice []uint16

func (p UInt16Slice) Len() int           { return len(p) }
func (p UInt16Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p UInt16Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

type Instance struct {
	Image        string           `json:"image"`
	Name         string           `json:"name"`
	Hostname     string           `json:"hostname"`
	IP           string           `json:"ip"`
	IsManager    *bool            `json:"is_manager"`
	Mem          string           `json:"mem"`
	Cpu          string           `json:"cpu"`
	Alias        string           `json:"alias"`
	ServerCert   []byte           `json:"server_cert"`
	ServerKey    []byte           `json:"server_key"`
	CACert       []byte           `json:"ca_cert"`
	Cert         []byte           `json:"cert"`
	Key          []byte           `json:"key"`
	IsDockerHost bool             `json:"is_docker_host"`
	Docker       docker.DockerApi `json:"-"`
	Session      *Session         `json:"-"`
	Terminal     net.Conn         `json:"-"`
	ctx          context.Context  `json:"-"`
	tempPorts    []uint16         `json:"-"`
	Ports        UInt16Slice
	rw           sync.Mutex
}

func (i *Instance) SetUsedPort(port uint16) {
	i.rw.Lock()
	defer i.rw.Unlock()

	for _, p := range i.tempPorts {
		if p == port {
			return
		}
	}
	i.tempPorts = append(i.tempPorts, port)
}
func (i *Instance) GetUsedPorts() []uint16 {
	i.rw.Lock()
	defer i.rw.Unlock()

	return i.tempPorts
}
func (i *Instance) CleanUsedPorts() {
	i.rw.Lock()
	defer i.rw.Unlock()

	i.tempPorts = []uint16{}
}
