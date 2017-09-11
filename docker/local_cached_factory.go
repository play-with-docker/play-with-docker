package docker

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/docker/docker/api"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/tlsconfig"
	"github.com/play-with-docker/play-with-docker/pwd/types"
	"github.com/play-with-docker/play-with-docker/router"
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

func (f *localCachedFactory) GetForSession(sessionId string) (DockerApi, error) {
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
	key := instance.SessionId + instance.IP

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

	proxyUrl, _ := url.Parse("http://l2:443")
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   1 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConnsPerHost: 5,
		Proxy:               http.ProxyURL(proxyUrl),
	}
	if tlsConfig != nil {
		transport.TLSClientConfig = tlsConfig
	}
	cli := &http.Client{
		Transport: transport,
	}
	dc, err := client.NewClient(fmt.Sprintf("http://%s", router.EncodeHost(instance.SessionId, instance.IP, router.HostOpts{EncodedPort: 2375})), api.DefaultVersion, cli, nil)
	if err != nil {
		return nil, fmt.Errorf("Could not connect to DinD docker daemon", err)
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
			log.Printf("Connection to [%s] has failed, maybe instance is not ready yet, sleeping and retrying in 1 second. Try #%d\n", c.DaemonHost(), i+1)
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
