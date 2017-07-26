package task

import (
	"context"
	"fmt"
	"time"

	"github.com/play-with-docker/play-with-docker/pwd/types"
	"github.com/play-with-docker/play-with-docker/storage"
)

type Task interface {
	Name() string
	Run(ctx context.Context, instance *types.Instance) error
}

type SchedulerApi interface {
	Schedule(session *types.Session) error
	Unschedule(session *types.Session) error
	Start()
	Stop()
	AddTask(task Task) error
	RemoveTask(task Task) error
}

type scheduledSession struct {
	session *types.Session
	cancel  context.CancelFunc
	ctx     context.Context
	ticker  *time.Ticker
}

type scheduler struct {
	scheduledSessions map[string]*scheduledSession
	storage           storage.StorageApi
	tasks             map[string]Task
	started           bool
}

func NewScheduler(s storage.StorageApi) (*scheduler, error) {
	sch := &scheduler{storage: s}

	sch.tasks = make(map[string]Task)

	err := sch.loadFromStorage()
	if err != nil {
		return nil, err
	}

	return sch, nil
}

func (s *scheduler) loadFromStorage() error {
	sessions, err := s.storage.SessionGetAll()
	if err != nil {
		return err
	}
	s.scheduledSessions = make(map[string]*scheduledSession, len(sessions))
	for _, session := range sessions {
		s.register(session)
	}

	return nil
}

func (s *scheduler) AddTask(task Task) error {
	if _, found := s.tasks[task.Name()]; found {
		return fmt.Errorf("Task [%s] was already added", task.Name())
	}
	s.tasks[task.Name()] = task

	return nil
}

func (s *scheduler) RemoveTask(task Task) error {
	if _, found := s.tasks[task.Name()]; !found {
		return fmt.Errorf("Task [%s] doesn't exist", task.Name())
	}
	delete(s.tasks, task.Name())

	return nil
}

func (s *scheduler) Start() {
	for _, session := range s.scheduledSessions {
		go s.cron(session)
	}
	s.started = true
}

func (s *scheduler) register(session *types.Session) *scheduledSession {
	ctx, cancel := context.WithCancel(context.Background())
	s.scheduledSessions[session.Id] = &scheduledSession{session: session, cancel: cancel, ctx: ctx}
	return s.scheduledSessions[session.Id]
}

func (s *scheduler) cron(session *scheduledSession) {
	session.ticker = time.NewTicker(1 * time.Second)

	for {
		select {
		case <-session.ticker.C:
			s.processSession(session.ctx, session.session)
		case <-session.ctx.Done():
			return
		}
	}
}

func (s *scheduler) processSession(ctx context.Context, session *types.Session) {
	for _, ins := range session.Instances {
		go s.processInstance(ctx, ins)
	}
}

func (s *scheduler) processInstance(ctx context.Context, instance *types.Instance) {
	for _, task := range s.tasks {
		task.Run(ctx, instance)
	}
}

func (s *scheduler) Schedule(session *types.Session) error {
	if !s.started {
		return fmt.Errorf("Can only schedule sessions after the scheduler has been started.")
	}
	if _, found := s.scheduledSessions[session.Id]; found {
		return fmt.Errorf("Session [%s] was already scheduled", session.Id)
	}
	go s.cron(s.register(session))
	return nil
}

func (s *scheduler) Unschedule(session *types.Session) error {
	if !s.started {
		return fmt.Errorf("Can only schedule sessions after the scheduler has been started.")
	}
	if _, found := s.scheduledSessions[session.Id]; !found {
		return fmt.Errorf("Session [%s] in not scheduled", session.Id)
	}

	scheduledSession := s.scheduledSessions[session.Id]
	scheduledSession.cancel()
	scheduledSession.ticker.Stop()
	delete(s.scheduledSessions, session.Id)

	return nil
}
