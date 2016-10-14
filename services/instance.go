package services

import (
	"log"
	"os"

	"github.com/franela/play-with-docker/types"
)

var dindImage string
var defaultDindImageName string

func init() {
	dindImage = getDindImageName()
}

func getDindImageName() string {
	dindImage := os.Getenv("DIND_IMAGE")
	defaultDindImageName = "docker:1.12.2-rc2-dind"
	if len(dindImage) == 0 {
		dindImage = defaultDindImageName
	}
	return dindImage
}

func NewInstance(session *types.Session) (*types.Instance, error) {

	//TODO: Validate that a session can only have 5 instances
	//TODO: Create in redis
	log.Printf("NewInstance - using image: [%s]\n", dindImage)
	instance, err := CreateInstance(session.Id, dindImage)

	if err != nil {
		return nil, err
	}

	if session.Instances == nil {
		session.Instances = make(map[string]*types.Instance)
	}
	session.Instances[instance.Name] = instance

	return instance, nil
}

func GetInstance(session *types.Session, name string) *types.Instance {
	//TODO: Use redis
	return session.Instances[name]
}
func DeleteInstance(session *types.Session, instance *types.Instance) error {
	//TODO: Use redis
	delete(session.Instances, instance.Name)
	return DeleteContainer(instance.Name)
}
