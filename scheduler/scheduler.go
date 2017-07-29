package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/play-with-docker/play-with-docker/event"
	"github.com/play-with-docker/play-with-docker/pwd"
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
	ticker  *time.Ticker
}

type scheduler struct {
	scheduledSessions map[string]*scheduledSession
	storage           storage.StorageApi
	event             event.EventApi
	pwd               pwd.PWDApi
	tasks             map[string]Task
	started           bool
}

func NewScheduler(s storage.StorageApi, e event.EventApi, p pwd.PWDApi) (*scheduler, error) {
	sch := &scheduler{storage: s, event: e, pwd: p}

	sch.tasks = make(map[string]Task)
	sch.scheduledSessions = make(map[string]*scheduledSession)

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

func (s *scheduler) Stop() {
	for _, session := range s.scheduledSessions {
		s.Unschedule(session.session)
	}
	s.started = false
}

func (s *scheduler) Start() {
	for _, session := range s.scheduledSessions {
		ctx, cancel := context.WithCancel(context.Background())
		session.cancel = cancel
		session.ticker = time.NewTicker(1 * time.Second)
		go s.cron(ctx, session)
	}
	s.event.On(event.SESSION_NEW, func(sessionId string, args ...interface{}) {
		session, err := s.storage.SessionGet(sessionId)
		if err != nil {
			log.Printf("Session [%s] was not found in storage. Got %s\n", sessionId, err)
			return
		}
		s.Schedule(session)
	})
	s.event.On(event.SESSION_END, func(sessionId string, args ...interface{}) {
		session, err := s.storage.SessionGet(sessionId)
		if err != nil {
			log.Printf("Session [%s] was not found in storage. Got %s\n", sessionId, err)
			return
		}
		err = s.Unschedule(session)
		if err != nil {
			log.Println(err)
			return
		}
	})
	s.started = true
}

func (s *scheduler) register(session *types.Session) *scheduledSession {
	ss := &scheduledSession{session: session}
	s.scheduledSessions[session.Id] = ss
	return ss
}

func (s *scheduler) cron(ctx context.Context, session *scheduledSession) {
	for {
		select {
		case <-session.ticker.C:
			if time.Now().After(session.session.ExpiresAt) {
				// Session has expired. Need to close the session.
				s.pwd.SessionClose(session.session)
				return
			} else {
				s.processSession(ctx, session.session)
			}
		case <-ctx.Done():
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
		go task.Run(ctx, instance)
	}
}

func (s *scheduler) Schedule(session *types.Session) error {
	if !s.started {
		return fmt.Errorf("Can only schedule sessions after the scheduler has been started.")
	}
	if _, found := s.scheduledSessions[session.Id]; found {
		return fmt.Errorf("Session [%s] was already scheduled", session.Id)
	}
	scheduledSession := s.register(session)
	ctx, cancel := context.WithCancel(context.Background())
	scheduledSession.cancel = cancel
	go s.cron(ctx, scheduledSession)
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
