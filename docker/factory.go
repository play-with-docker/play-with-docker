package docker

type FactoryApi interface {
	GetForSession(sessionId string) (DockerApi, error)
	GetForInstance(sessionId, instanceName string) (DockerApi, error)
}
