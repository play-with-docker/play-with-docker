package task

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"

	dockerTypes "docker.io/go-docker/api/types"
	"github.com/play-with-docker/play-with-docker/docker"
	"github.com/play-with-docker/play-with-docker/event"
	"github.com/play-with-docker/play-with-docker/pwd/types"
	"github.com/play-with-docker/play-with-docker/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockSessionProvider struct {
	mock.Mock
}

func (m *mockSessionProvider) GetDocker(session *types.Session) (docker.DockerApi, error) {
	args := m.Called(session)

	return args.Get(0).(docker.DockerApi), args.Error(1)
}

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }

func TestCollectStats_Name(t *testing.T) {
	e := &event.Mock{}
	f := &docker.FactoryMock{}
	s := &storage.Mock{}

	task := NewCollectStats(e, f, s)

	assert.Equal(t, "CollectStats", task.Name())
	e.M.AssertExpectations(t)
	f.AssertExpectations(t)
}

func TestCollectStats_Run(t *testing.T) {
	d := &docker.Mock{}
	e := &event.Mock{}
	f := &docker.FactoryMock{}
	s := &storage.Mock{}

	stats := dockerTypes.StatsJSON{}
	b, _ := json.Marshal(stats)
	i := &types.Instance{
		IP:        "10.0.0.1",
		Name:      "aaaabbbb_node1",
		SessionId: "aaaabbbbcccc",
		Hostname:  "node1",
	}

	sess := &types.Session{
		Id: "aaaabbbbcccc",
	}

	s.On("SessionGet", i.SessionId).Return(sess, nil)
	f.On("GetForSession", sess).Return(d, nil)
	d.On("ContainerStats", i.Name).Return(nopCloser{bytes.NewReader(b)}, nil)
	e.M.On("Emit", CollectStatsEvent, "aaaabbbbcccc", []interface{}{InstanceStats{Instance: i.Name, Mem: "0.00% (0B / 0B)", Cpu: "0.00%"}}).Return()

	task := NewCollectStats(e, f, s)
	ctx := context.Background()

	err := task.Run(ctx, i)

	assert.Nil(t, err)
	d.AssertExpectations(t)
	e.M.AssertExpectations(t)
	f.AssertExpectations(t)
}
