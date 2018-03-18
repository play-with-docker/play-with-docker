package docker

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	client "docker.io/go-docker"
	"docker.io/go-docker/api"
	"github.com/docker/go-connections/tlsconfig"
	"github.com/play-with-docker/play-with-docker/pwd/types"
	"github.com/play-with-docker/play-with-docker/router"
)

type FactoryApi interface {
	GetForSession(session *types.Session) (DockerApi, error)
	GetForInstance(instance *types.Instance) (DockerApi, error)
}

func NewClient(instance *types.Instance, proxyHost string) (*client.Client, error) {
	var host string
	var durl string

	var tlsConfig *tls.Config
	if (len(instance.Cert) > 0 && len(instance.Key) > 0) || instance.Tls {
		host = router.EncodeHost(instance.SessionId, instance.RoutableIP, router.HostOpts{EncodedPort: 2376})
		tlsConfig = tlsconfig.ClientDefault()
		tlsConfig.InsecureSkipVerify = true
		tlsConfig.ServerName = host
		if len(instance.Cert) > 0 && len(instance.Key) > 0 {
			tlsCert, err := tls.X509KeyPair(instance.Cert, instance.Key)
			if err != nil {
				return nil, fmt.Errorf("Could not load X509 key pair: %v. Make sure the key is not encrypted", err)
			}
			tlsConfig.Certificates = []tls.Certificate{tlsCert}
		}
	} else {
		host = router.EncodeHost(instance.SessionId, instance.RoutableIP, router.HostOpts{EncodedPort: 2375})
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   1 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConnsPerHost: 5,
	}

	if tlsConfig != nil {
		transport.TLSClientConfig = tlsConfig
		durl = fmt.Sprintf("https://%s", proxyHost)
	} else {
		transport.Proxy = http.ProxyURL(&url.URL{Host: proxyHost})
		durl = fmt.Sprintf("http://%s", host)
	}

	cli := &http.Client{
		Transport: transport,
	}

	dc, err := client.NewClient(durl, api.DefaultVersion, cli, nil)
	if err != nil {
		return nil, fmt.Errorf("Could not connect to DinD docker daemon: %v", err)
	}

	return dc, nil
}
