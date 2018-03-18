package k8s

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/docker/go-connections/tlsconfig"
	"github.com/play-with-docker/play-with-docker/pwd/types"
	"github.com/play-with-docker/play-with-docker/router"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

type FactoryApi interface {
	GetForInstance(instance *types.Instance) (*kubernetes.Clientset, error)
	GetKubeletForInstance(instance *types.Instance) (*KubeletClient, error)
}

func NewClient(instance *types.Instance, proxyHost string) (*kubernetes.Clientset, error) {
	var durl string

	host := router.EncodeHost(instance.SessionId, instance.RoutableIP, router.HostOpts{EncodedPort: 6443})

	var tlsConfig *tls.Config
	tlsConfig = tlsconfig.ClientDefault()
	tlsConfig.InsecureSkipVerify = true
	tlsConfig.ServerName = host

	var transport http.RoundTripper
	transport = &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   1 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSClientConfig:     tlsConfig,
		MaxIdleConnsPerHost: 5,
	}

	durl = fmt.Sprintf("https://%s", proxyHost)

	cc := rest.ContentConfig{
		ContentType:          "application/json",
		GroupVersion:         &schema.GroupVersion{Version: "v1"},
		NegotiatedSerializer: serializer.DirectCodecFactory{CodecFactory: scheme.Codecs},
	}
	restConfig := &rest.Config{
		Host:          durl,
		APIPath:       "/api/",
		BearerToken:   "31ada4fd-adec-460c-809a-9e56ceb75269",
		ContentConfig: cc,
	}

	transport, err := rest.HTTPWrappersForConfig(restConfig, transport)
	if err != nil {
		return nil, fmt.Errorf("Error wrapping transport %v", err)
	}
	cli := &http.Client{
		Transport: transport,
	}

	rc, err := rest.RESTClientFor(restConfig)
	rc.Client = cli
	if err != nil {
		return nil, fmt.Errorf("Error creating K8s client %v", err)
	}

	return kubernetes.New(rc), nil
}

func NewKubeletClient(instance *types.Instance, proxyHost string) (*KubeletClient, error) {
	var durl string

	host := router.EncodeHost(instance.SessionId, instance.RoutableIP, router.HostOpts{EncodedPort: 10255})

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   1 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConnsPerHost: 5,
	}

	durl = fmt.Sprintf("http://%s", host)
	transport.Proxy = http.ProxyURL(&url.URL{Host: proxyHost})

	cli := &http.Client{
		Transport: transport,
	}
	kc := &KubeletClient{client: cli, baseURL: durl}
	return kc, nil
}

type KubeletClient struct {
	client  *http.Client
	baseURL string
}

func (c *KubeletClient) Get(path string) (*http.Response, error) {
	return c.client.Get(c.baseURL + path)
}

type metadata struct {
	Labels map[string]string
}

type item struct {
	Metadata metadata
}

type kubeletPodsResponse struct {
	Items []item
}

func (c *KubeletClient) IsManager() (bool, error) {
	res, err := c.client.Get(c.baseURL + "/pods")
	if err != nil {
		return false, err
	}
	podsData := &kubeletPodsResponse{}

	json.NewDecoder(res.Body).Decode(podsData)

	for _, i := range podsData.Items {
		for _, v := range i.Metadata.Labels {
			if v == "kube-apiserver" {
				return true, nil
			}
		}
	}

	return false, nil
}
