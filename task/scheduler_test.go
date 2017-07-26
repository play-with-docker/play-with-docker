package task

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"testing"

	"github.com/play-with-docker/play-with-docker/pwd/types"
	"github.com/play-with-docker/play-with-docker/storage"
	"github.com/stretchr/testify/assert"
)

type mockTask struct {
	name string
	run  func(ctx context.Context, instance *types.Instance) error
}

func (m *mockTask) Name() string {
	return m.name
}
func (m *mockTask) Run(ctx context.Context, instance *types.Instance) error {
	return m.run(ctx, instance)
}

func mockStorage() storage.StorageApi {
	tmpfile, err := ioutil.TempFile("", "pwd")
	if err != nil {
		log.Fatal(err)
	}
	tmpfile.Close()
	os.Remove(tmpfile.Name())
	store, _ := storage.NewFileStorage(tmpfile.Name())
	return store
}

func TestNew(t *testing.T) {
	store := mockStorage()

	s := &types.Session{
		Id: "aaabbbccc",
		Instances: map[string]*types.Instance{
			"node1": &types.Instance{
				Name: "node1",
				IP:   "10.0.0.1",
			},
		},
	}
	err := store.SessionPut(s)
	assert.Nil(t, err)

	sch, err := NewScheduler(store)
	assert.Nil(t, err)
	assert.Len(t, sch.scheduledSessions, 1)
}

func TestAddTask(t *testing.T) {
	store := mockStorage()
	sch, err := NewScheduler(store)
	assert.Nil(t, err)

	task := &mockTask{name: "FooBar"}
	err = sch.AddTask(task)
	assert.Nil(t, err)

	err = sch.AddTask(task)
	assert.NotNil(t, err)

	assert.Equal(t, map[string]Task{"FooBar": task}, sch.tasks)
}

func TestRemoveTask(t *testing.T) {
	store := mockStorage()
	sch, err := NewScheduler(store)
	assert.Nil(t, err)

	task := &mockTask{name: "FooBar"}
	err = sch.AddTask(task)
	assert.Nil(t, err)

	err = sch.RemoveTask(task)
	assert.Nil(t, err)

	err = sch.RemoveTask(task)
	assert.NotNil(t, err)

	assert.Equal(t, map[string]Task{}, sch.tasks)
}

func TestStart(t *testing.T) {
	store := mockStorage()

	s := &types.Session{
		Id: "aaabbbccc",
		Instances: map[string]*types.Instance{
			"node1": &types.Instance{
				Name: "node1",
				IP:   "10.0.0.1",
			},
		},
	}
	err := store.SessionPut(s)
	assert.Nil(t, err)

	sch, err := NewScheduler(store)
	assert.Nil(t, err)

	wg := sync.WaitGroup{}
	wg.Add(1)
	ran := false
	task := &mockTask{name: "FooBar", run: func(ctx context.Context, instance *types.Instance) error {
		ran = true
		wg.Done()
		return nil
	}}
	err = sch.AddTask(task)
	assert.Nil(t, err)

	sch.Start()
	wg.Wait()
	assert.True(t, ran)
}
