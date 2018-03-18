package id

import "github.com/stretchr/testify/mock"

type MockGenerator struct {
	mock.Mock
}

func (m *MockGenerator) NewId() string {
	args := m.Called()
	return args.String(0)
}
