package types

type Session struct {
	Id        string               `json:"id"`
	Instances map[string]*Instance `json:"instances"`
}

type Instance struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
}
