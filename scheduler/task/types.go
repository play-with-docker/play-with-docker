package task

type ClusterStatus struct {
	IsManager bool   `json:"is_manager"`
	IsWorker  bool   `json:"is_worker"`
	Instance  string `json:"instance"`
}

type ClusterPorts struct {
	Manager   string   `json:"manager"`
	Instances []string `json:"instances"`
	Ports     []int    `json:"ports"`
}
