package docker

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	client "docker.io/go-docker"
	"github.com/play-with-docker/play-with-docker/pwd/types"
	"github.com/play-with-docker/play-with-docker/storage"
)

type localCachedFactory struct {
	rw              sync.Mutex
	irw             sync.Mutex
	sessionClient   DockerApi
	instanceClients map[string]*instanceEntry
	storage         storage.StorageApi
}

type instanceEntry struct {
	rw     sync.Mutex
	client DockerApi
}

func (f *localCachedFactory) GetForSession(session *types.Session) (DockerApi, error) {
	f.rw.Lock()
	defer f.rw.Unlock()

	if f.sessionClient != nil {
		if err := f.check(f.sessionClient.GetClient()); err == nil {
			return f.sessionClient, nil
		} else {
			f.sessionClient.GetClient().Close()
		}
	}

	c, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}
	err = f.check(c)
	if err != nil {
		return nil, err
	}
	d := NewDocker(c)
	f.sessionClient = d
	return f.sessionClient, nil
}

func (f *localCachedFactory) GetForInstance(instance *types.Instance) (DockerApi, error) {
	key := instance.Name

	f.irw.Lock()
	c, found := f.instanceClients[key]
	if !found {
		c := &instanceEntry{}
		f.instanceClients[key] = c
	}
	c = f.instanceClients[key]
	f.irw.Unlock()

	c.rw.Lock()
	defer c.rw.Unlock()

	if c.client != nil {
		if err := f.check(c.client.GetClient()); err == nil {
			return c.client, nil
		} else {
			c.client.GetClient().Close()
		}
	}

	dc, err := NewClient(instance, "l2:443")
	if err != nil {
		return nil, err
	}
	err = f.check(dc)
	if err != nil {
		return nil, err
	}
	dockerClient := NewDocker(dc)
	c.client = dockerClient

	return dockerClient, nil
}

func (f *localCachedFactory) check(c *client.Client) error {
	ok := false
	for i := 0; i < 5; i++ {
		_, err := c.Ping(context.Background())
		if err != nil {
			log.Printf("Connection to [%s] has failed, maybe instance is not ready yet, sleeping and retrying in 1 second. Try #%d. Got: %v\n", c.DaemonHost(), i+1, err)
			time.Sleep(time.Second)
			continue
		}
		ok = true
		break
	}
	if !ok {
		return fmt.Errorf("Connection to docker daemon was not established.")
	}
	return nil
}

func NewLocalCachedFactory(s storage.StorageApi) *localCachedFactory {
	return &localCachedFactory{
		instanceClients: make(map[string]*instanceEntry),
		storage:         s,
	}
}
