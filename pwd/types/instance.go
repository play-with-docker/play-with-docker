package types

import (
	"context"
	"sync"
)

type Instance struct {
	Image        string          `json:"image" bson:"image"`
	Name         string          `json:"name" bson:"name"`
	Hostname     string          `json:"hostname" bson:"hostname"`
	IP           string          `json:"ip" bson:"ip"`
	ServerCert   []byte          `json:"server_cert" bson:"server_cert"`
	ServerKey    []byte          `json:"server_key" bson:"server_key"`
	CACert       []byte          `json:"ca_cert" bson:"ca_cert"`
	Cert         []byte          `json:"cert" bson:"cert"`
	Key          []byte          `json:"key" bson:"key"`
	IsDockerHost bool            `json:"is_docker_host" bson:"is_docker_host"`
	SessionId    string          `json:"session_id" bson:"session_id"`
	Proxy        string          `json:"proxy" bson:"proxy"`
	Session      *Session        `json:"-" bson:"-"`
	ctx          context.Context `json:"-" bson:"-"`
	rw           sync.Mutex
}
