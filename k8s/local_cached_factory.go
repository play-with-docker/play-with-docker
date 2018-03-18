package k8s

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/play-with-docker/play-with-docker/pwd/types"
	"github.com/play-with-docker/play-with-docker/storage"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type localCachedFactory struct {
	rw              sync.Mutex
	irw             sync.Mutex
	sessionClient   *kubernetes.Clientset
	instanceClients map[string]*instanceEntry
	storage         storage.StorageApi
}

type instanceEntry struct {
	rw            sync.Mutex
	client        *kubernetes.Clientset
	kubeletClient *KubeletClient
}

func (f *localCachedFactory) GetForInstance(instance *types.Instance) (*kubernetes.Clientset, error) {
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

	if c.client == nil {
		kc, err := NewClient(instance, "l2:443")
		if err != nil {
			return nil, err
		}
		c.client = kc
	}

	err := f.check(func() error {
		_, err := c.client.CoreV1().Pods("").List(metav1.ListOptions{})
		return err
	})
	if err != nil {
		return nil, err
	}

	return c.client, nil
}

func (f *localCachedFactory) GetKubeletForInstance(instance *types.Instance) (*KubeletClient, error) {
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

	if c.kubeletClient == nil {
		kc, err := NewKubeletClient(instance, "l2:443")
		if err != nil {
			return nil, err
		}
		c.kubeletClient = kc
	}

	err := f.check(func() error {
		r, err := c.kubeletClient.Get("/pods")
		if err != nil {
			return err
		}
		defer r.Body.Close()
		return nil
	})
	if err != nil {
		return nil, err
	}

	return c.kubeletClient, nil
}

func (f *localCachedFactory) check(fn func() error) error {
	ok := false
	for i := 0; i < 5; i++ {
		err := fn()
		if err != nil {
			log.Printf("Connection to k8s api has failed, maybe instance is not ready yet, sleeping and retrying in 1 second. Try #%d. Got: %v\n", i+1, err)
			time.Sleep(time.Second)
			continue
		}
		ok = true
		break
	}
	if !ok {
		return fmt.Errorf("Connection to k8s api was not established.")
	}
	return nil
}

func NewLocalCachedFactory(s storage.StorageApi) *localCachedFactory {
	return &localCachedFactory{
		instanceClients: make(map[string]*instanceEntry),
		storage:         s,
	}
}
