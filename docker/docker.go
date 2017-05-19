package docker

type Docker interface {
	CreateNetwork(id string) error
}
