package types

import (
	"context"
	"sync"

	"github.com/play-with-docker/play-with-docker/docker"
)

type UInt16Slice []uint16

func (p UInt16Slice) Len() int           { return len(p) }
func (p UInt16Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p UInt16Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

type Instance struct {
	Image         string           `json:"image" bson:"image"`
	Name          string           `json:"name" bson:"name"`
	Hostname      string           `json:"hostname" bson:"hostname"`
	IP            string           `json:"ip" bson:"ip"`
	IsManager     *bool            `json:"is_manager" bson:"is_manager"`
	Mem           string           `json:"mem" bson:"mem"`
	Cpu           string           `json:"cpu" bson:"cpu"`
	Alias         string           `json:"alias" bson:"alias"`
	ServerCert    []byte           `json:"server_cert" bson:"server_cert"`
	ServerKey     []byte           `json:"server_key" bson:"server_key"`
	CACert        []byte           `json:"ca_cert" bson:"ca_cert"`
	Cert          []byte           `json:"cert" bson:"cert"`
	Key           []byte           `json:"key" bson:"key"`
	IsDockerHost  bool             `json:"is_docker_host" bson:"is_docker_host"`
	SessionId     string           `json:"session_id" bson:"session_id"`
	SessionPrefix string           `json:"session_prefix" bson:"session_prefix"`
	Docker        docker.DockerApi `json:"-"`
	Session       *Session         `json:"-" bson:"-"`
	ctx           context.Context  `json:"-" bson:"-"`
	tempPorts     []uint16         `json:"-" bson:"-"`
	Ports         UInt16Slice
	rw            sync.Mutex
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
