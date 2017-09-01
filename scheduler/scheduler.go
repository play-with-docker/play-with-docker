package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
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
	busy    int32
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
		session := &types.Session{Id: sessionId}
		err := s.Unschedule(session)
		if err != nil {
			log.Println(err)
			return
		}
	})
	s.started = true
}

func (s *scheduler) register(session *types.Session) *scheduledSession {
	ss := &scheduledSession{session: session, busy: 0}
	s.scheduledSessions[session.Id] = ss
	return ss
}

func (s *scheduler) cron(ctx context.Context, session *scheduledSession) {
	for {
		select {
		case <-session.ticker.C:
			if atomic.CompareAndSwapInt32(&session.busy, 0, 1) {
				if time.Now().After(session.session.ExpiresAt) {
					// Session has expired. Need to close the session.
					s.pwd.SessionClose(session.session)
					return
				} else {
					s.processSession(ctx, session.session)
				}
				atomic.StoreInt32(&session.busy, 0)
			} else {
				log.Printf("Session [%s] is currently busy. Will try next time.\n", session.session.Id)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *scheduler) processSession(ctx context.Context, session *types.Session) {
	updatedSession, err := s.storage.SessionGet(session.Id)
	if err != nil {
		if storage.NotFound(err) {
			log.Printf("Session [%s] was not found in storage. Unscheduling.\n", session.Id)
			s.Unschedule(session)
		} else {
			log.Printf("Cannot process session. Got %s\n", err)
		}
		return
	}

	instances, err := s.storage.InstanceFindBySessionId(updatedSession.Id)
	if err != nil {
		log.Printf("Couldn't find instances for session [%s]. Got: %v\n", updatedSession.Id, err)
		return
	}
	wg := sync.WaitGroup{}
	wg.Add(len(instances))
	for _, ins := range instances {
		go func(ins *types.Instance) {
			s.processInstance(ctx, ins)
			wg.Done()
		}(ins)
	}
	wg.Wait()
}

func (s *scheduler) processInstance(ctx context.Context, instance *types.Instance) {
	wg := sync.WaitGroup{}
	wg.Add(len(s.tasks))
	for _, task := range s.tasks {
		go func(task Task) {
			task.Run(ctx, instance)
			wg.Done()
		}(task)
	}
	wg.Wait()
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
	scheduledSession.ticker = time.NewTicker(1 * time.Second)
	go s.cron(ctx, scheduledSession)

	log.Printf("Scheduled session [%s]\n", session.Id)

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

	log.Printf("Unscheduled session [%s]\n", session.Id)

	return nil
}
