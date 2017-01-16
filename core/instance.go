package core

type Instance struct {
	Name       string   `json:"name"`
	Hostname   string   `json:"hostname"`
	IP         string   `json:"ip"`
	IsManager  *bool    `json:"is_manager"`
	Mem        string   `json:"mem"`
	Cpu        string   `json:"cpu"`
	Ports      []uint16 `json:"ports"`
	ServerCert []byte   `json:"server_cert"`
	ServerKey  []byte   `json:"server_key"`
}
