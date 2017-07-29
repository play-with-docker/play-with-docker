package docker

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/docker/docker/api"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/tlsconfig"
	"github.com/play-with-docker/play-with-docker/router"
	"github.com/play-with-docker/play-with-docker/storage"
)

type localCachedFactory struct {
	rw              sync.Mutex
	sessionClient   DockerApi
	instanceClients map[string]DockerApi
	storage         storage.StorageApi
}

func (f *localCachedFactory) GetForSession(sessionId string) (DockerApi, error) {
	f.rw.Lock()
	defer f.rw.Unlock()

	if f.sessionClient != nil {
		return f.sessionClient, nil
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

func (f *localCachedFactory) GetForInstance(sessionId, instanceName string) (DockerApi, error) {
	f.rw.Lock()
	defer f.rw.Unlock()

	c, found := f.instanceClients[sessionId+instanceName]
	if found {
		return c, nil
	}

	instance, err := f.storage.InstanceGet(sessionId, instanceName)
	if err != nil {
		return nil, err
	}
	// Need to create client to the DinD docker daemon
	// We check if the client needs to use TLS
	var tlsConfig *tls.Config
	if len(instance.Cert) > 0 && len(instance.Key) > 0 {
		tlsConfig = tlsconfig.ClientDefault()
		tlsConfig.InsecureSkipVerify = true
		tlsCert, err := tls.X509KeyPair(instance.Cert, instance.Key)
		if err != nil {
			return nil, fmt.Errorf("Could not load X509 key pair: %v. Make sure the key is not encrypted", err)
		}
		tlsConfig.Certificates = []tls.Certificate{tlsCert}
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   1 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext}
	if tlsConfig != nil {
		transport.TLSClientConfig = tlsConfig
	}
	cli := &http.Client{
		Transport: transport,
	}
	dc, err := client.NewClient("http://192.168.1.5:443", api.DefaultVersion, cli, map[string]string{"X-Forwarded-Host": router.EncodeHost(instance.SessionId, instance.IP, router.HostOpts{EncodedPort: 2375})})
	if err != nil {
		return nil, fmt.Errorf("Could not connect to DinD docker daemon", err)
	}
	err = f.check(dc)
	if err != nil {
		return nil, err
	}
	dockerClient := NewDocker(dc)
	f.instanceClients[sessionId+instance.Name] = dockerClient

	return dockerClient, nil
}

func (f *localCachedFactory) check(c *client.Client) error {
	ok := false
	for i := 0; i < 5; i++ {
		_, err := c.Ping(context.Background())
		if err != nil {
			if client.IsErrConnectionFailed(err) {
				// connection has failed, maybe instance is not ready yet, sleep and retry
				log.Printf("Connection to [%s] has failed, maybe instance is not ready yet, sleeping and retrying in 1 second. Try #%d\n", c.DaemonHost(), i+1)
				time.Sleep(time.Second)
				continue
			}
			return err
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
		instanceClients: make(map[string]DockerApi),
		storage:         s,
	}
}
