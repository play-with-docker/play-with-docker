package scheduler

import (
	"context"
	"testing"

	"github.com/play-with-docker/play-with-docker/event"
	"github.com/play-with-docker/play-with-docker/pwd"
	"github.com/play-with-docker/play-with-docker/pwd/types"
	"github.com/play-with-docker/play-with-docker/storage"
	"github.com/stretchr/testify/assert"
)

type fakeTask struct {
	name string
}

func (f fakeTask) Name() string {
	return f.name
}
func (f fakeTask) Run(ctx context.Context, instance *types.Instance) error {
	return nil
}

func TestScheduler_getMatchedTasks(t *testing.T) {
	tasks := []Task{
		fakeTask{name: "docker_task1"},
		fakeTask{name: "docker_task2"},
		fakeTask{name: "k8s_task1"},
		fakeTask{name: "k8s_task2"},
	}

	_s := &storage.Mock{}
	_e := &event.Mock{}
	_p := &pwd.Mock{}

	s, err := NewScheduler(tasks, _s, _e, _p)
	assert.Nil(t, err)

	// No matches
	matched := s.getMatchedTasks(&types.Playground{Tasks: []string{}})
	assert.Empty(t, matched)

	// Match everything
	matched = s.getMatchedTasks(&types.Playground{Tasks: []string{".*"}})
	assert.Subset(t, tasks, matched)
	assert.Len(t, matched, len(tasks))

	// Match some
	matched = s.getMatchedTasks(&types.Playground{Tasks: []string{"docker_.*"}})
	assert.Subset(t, []Task{fakeTask{name: "docker_task1"}, fakeTask{name: "docker_task2"}}, matched)
	assert.Len(t, matched, 2)

	// Match exactly
	matched = s.getMatchedTasks(&types.Playground{Tasks: []string{"docker_task1", "docker_task3"}})
	assert.Subset(t, []Task{fakeTask{name: "docker_task1"}}, matched)
	assert.Len(t, matched, 1)
}
