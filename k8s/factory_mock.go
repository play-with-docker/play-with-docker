package k8s

import (
	"github.com/play-with-docker/play-with-docker/pwd/types"
	"github.com/stretchr/testify/mock"
	"k8s.io/client-go/kubernetes"
)

type FactoryMock struct {
	mock.Mock
}

func (m *FactoryMock) GetKubeletForInstance(i *types.Instance) (*KubeletClient, error) {
	args := m.Called(i)
	return args.Get(0).(*KubeletClient), args.Error(1)
}

func (m *FactoryMock) GetForInstance(instance *types.Instance) (*kubernetes.Clientset, error) {
	args := m.Called(instance)
	return args.Get(0).(*kubernetes.Clientset), args.Error(1)
}
