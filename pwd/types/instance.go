package types

import "context"

type Instance struct {
	Name        string          `json:"name" bson:"name"`
	Image       string          `json:"image" bson:"image"`
	Hostname    string          `json:"hostname" bson:"hostname"`
	IP          string          `json:"ip" bson:"ip"`
	RoutableIP  string          `json:"routable_ip" bson:"routable_id"`
	ServerCert  []byte          `json:"server_cert" bson:"server_cert"`
	ServerKey   []byte          `json:"server_key" bson:"server_key"`
	CACert      []byte          `json:"ca_cert" bson:"ca_cert"`
	Cert        []byte          `json:"cert" bson:"cert"`
	Key         []byte          `json:"key" bson:"key"`
	Tls         bool            `json:"tls" bson:"tls"`
	SessionId   string          `json:"session_id" bson:"session_id"`
	ProxyHost   string          `json:"proxy_host" bson:"proxy_host"`
	SessionHost string          `json:"session_host" bson:"session_host"`
	Type        string          `json:"type" bson:"type"`
	WindowsId   string          `json:"-" bson:"windows_id"`
	ctx         context.Context `json:"-" bson:"-"`
}

type WindowsInstance struct {
	Id        string `bson:"id"`
	SessionId string `bson:"session_id"`
}

type InstanceConfig struct {
	ImageName      string
	Hostname       string
	ServerCert     []byte
	ServerKey      []byte
	CACert         []byte
	Cert           []byte
	Key            []byte
	Tls            bool
	PlaygroundFQDN string
	Type           string
}
