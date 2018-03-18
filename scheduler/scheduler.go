package scheduler

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sync"
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
	Start() error
	Stop()
}

type scheduledSession struct {
	session *types.Session
	cancel  context.CancelFunc
}

type scheduledInstance struct {
	instance     *types.Instance
	playgroundId string
	ticker       *time.Ticker
	cancel       context.CancelFunc
	fails        int
}

type scheduler struct {
	scheduledSessions  map[string]*scheduledSession
	scheduledInstances map[string]*scheduledInstance
	tasks              map[string]Task
	playgrounds        map[string]*types.Playground
	playgroundTasks    map[string][]Task
	started            bool
	ticker             *time.Ticker

	storage storage.StorageApi
	event   event.EventApi
	pwd     pwd.PWDApi
	mx      sync.Mutex
}

func NewScheduler(tasks []Task, s storage.StorageApi, e event.EventApi, p pwd.PWDApi) (*scheduler, error) {
	sch := &scheduler{storage: s, event: e, pwd: p}

	sch.tasks = make(map[string]Task)
	sch.scheduledSessions = make(map[string]*scheduledSession)
	sch.scheduledInstances = make(map[string]*scheduledInstance)
	sch.playgrounds = make(map[string]*types.Playground)
	sch.playgroundTasks = make(map[string][]Task)

	for _, task := range tasks {
		if err := sch.addTask(task); err != nil {
			return nil, err
		}
	}

	return sch, nil
}

func (s *scheduler) updatePlaygrounds() {
	s.mx.Lock()
	defer s.mx.Unlock()

	log.Printf("Updating playgrounds configuration\n")
	for playgroundId, _ := range s.playgrounds {
		playground, err := s.storage.PlaygroundGet(playgroundId)
		if err != nil {
			log.Printf("Could not find playground %s\n", playgroundId)
			continue
		}
		s.playgrounds[playgroundId] = playground
		matchedTasks := s.getMatchedTasks(playground)
		s.playgroundTasks[playground.Id] = matchedTasks
	}
}

func (s *scheduler) schedulePlaygroundsUpdate() {
	s.updatePlaygrounds()
	s.ticker = time.NewTicker(time.Minute * 5)
	go func() {
		for range s.ticker.C {
			s.updatePlaygrounds()
		}
	}()
}

func (s *scheduler) getMatchedTasks(playground *types.Playground) []Task {
	matchedTasks := []Task{}
	for _, expr := range playground.Tasks {
		for _, task := range s.tasks {
			if expr == task.Name() {
				matchedTasks = append(matchedTasks, task)
				continue
			}
			matched, err := regexp.MatchString(expr, task.Name())
			if err != nil {
				continue
			}
			if matched {
				matchedTasks = append(matchedTasks, task)
				continue
			}
		}
	}
	return matchedTasks
}

func (s *scheduler) getTasks(playgroundId string) []Task {
	s.mx.Lock()
	defer s.mx.Unlock()

	return s.playgroundTasks[playgroundId]
}

func (s *scheduler) processSession(ctx context.Context, ss *scheduledSession) {
	defer s.unscheduleSession(ss.session)
	select {
	case <-time.After(time.Until(ss.session.ExpiresAt)):
		// Session has expired. Need to close the session.
		s.pwd.SessionClose(ss.session)
		return
	case <-ctx.Done():
		return
	}
}
func (s *scheduler) processInstance(ctx context.Context, si *scheduledInstance) {
	defer s.unscheduleInstance(si.instance)
	for {
		select {
		case <-ctx.Done():
			log.Printf("Processing tasks for instance %s has been canceled.\n", si.instance.Name)
			return
		default:
			select {
			case <-si.ticker.C:
				// First check if instance still exists
				_, err := s.storage.InstanceGet(si.instance.Name)
				if err != nil {
					if storage.NotFound(err) {
						// Instance doesn't exists anymore. Unschedule.
						log.Printf("Instance %s doesn't exists in storage.\n", si.instance.Name)
						return
					}
					log.Printf("Error retrieving instance %s from storage. Got: %v\n", si.instance.Name, err)
					continue
				}
				for _, task := range s.getTasks(si.playgroundId) {
					err := task.Run(ctx, si.instance)
					if err != nil {
						log.Printf("Error running task %s on instance %s. Got: %v\n", task.Name(), si.instance.Name, err)
					}
				}
			}
		}
	}
}

func (s *scheduler) addTask(task Task) error {
	if _, found := s.tasks[task.Name()]; found {
		return fmt.Errorf("Task [%s] was already added", task.Name())
	}
	s.tasks[task.Name()] = task

	return nil
}

func (s *scheduler) unscheduleSession(session *types.Session) {
	ss, found := s.scheduledSessions[session.Id]
	if !found {
		return
	}

	ss.cancel()
	delete(s.scheduledSessions, ss.session.Id)
	log.Printf("Unscheduled session %s\n", session.Id)
}
func (s *scheduler) scheduleSession(session *types.Session) {
	if _, found := s.scheduledSessions[session.Id]; found {
		log.Printf("Session %s is already scheduled. Ignoring.\n", session.Id)
		return
	}
	ss := &scheduledSession{session: session}
	s.scheduledSessions[session.Id] = ss
	ctx, cancel := context.WithCancel(context.Background())
	ss.cancel = cancel
	go s.processSession(ctx, ss)
	log.Printf("Scheduled session %s\n", session.Id)
}
func (s *scheduler) unscheduleInstance(instance *types.Instance) {
	si, found := s.scheduledInstances[instance.Name]
	if !found {
		return
	}
	si.cancel()
	si.ticker.Stop()
	delete(s.scheduledInstances, si.instance.Name)
	log.Printf("Unscheduled instance %s\n", instance.Name)
}
func (s *scheduler) scheduleInstance(instance *types.Instance, playgroundId string) {
	if _, found := s.scheduledInstances[instance.Name]; found {
		log.Printf("Instance %s is already scheduled. Ignoring.\n", instance.Name)
		return
	}
	si := &scheduledInstance{instance: instance, playgroundId: playgroundId}
	s.scheduledInstances[instance.Name] = si
	ctx, cancel := context.WithCancel(context.Background())
	si.cancel = cancel
	si.ticker = time.NewTicker(time.Second)
	go s.processInstance(ctx, si)
	log.Printf("Scheduled instance %s\n", instance.Name)
}

func (s *scheduler) Stop() {
	s.ticker.Stop()
	for _, ss := range s.scheduledSessions {
		s.unscheduleSession(ss.session)
	}
	for _, si := range s.scheduledInstances {
		s.unscheduleInstance(si.instance)
	}
	s.started = false
}

func (s *scheduler) Start() error {
	sessions, err := s.storage.SessionGetAll()
	if err != nil {
		return err
	}
	for _, session := range sessions {
		s.scheduleSession(session)
		if _, found := s.playgrounds[session.PlaygroundId]; !found {
			playground, err := s.storage.PlaygroundGet(session.PlaygroundId)
			if err != nil {
				return err
			}
			s.playgrounds[playground.Id] = playground
		}

		instances, err := s.storage.InstanceFindBySessionId(session.Id)
		if err != nil {
			return err
		}

		for _, instance := range instances {
			s.scheduleInstance(instance, session.PlaygroundId)
		}
	}

	// Refresh playground conf every 5 minutes
	s.schedulePlaygroundsUpdate()

	s.event.On(event.SESSION_NEW, func(sessionId string, args ...interface{}) {
		s.mx.Lock()
		defer s.mx.Unlock()

		log.Printf("EVENT: Session New %s\n", sessionId)
		session, err := s.storage.SessionGet(sessionId)
		if err != nil {
			log.Printf("Session [%s] was not found in storage. Got %s\n", sessionId, err)
			return
		}
		if _, found := s.playgrounds[session.PlaygroundId]; !found {
			playground, err := s.storage.PlaygroundGet(session.PlaygroundId)
			if err != nil {
				log.Printf("Could not find playground %s\n", session.PlaygroundId)
				return
			}
			s.playgrounds[playground.Id] = playground
		}
		s.scheduleSession(session)
	})
	s.event.On(event.SESSION_END, func(sessionId string, args ...interface{}) {
		log.Printf("EVENT: Session End %s\n", sessionId)
		session := &types.Session{Id: sessionId}
		s.unscheduleSession(session)
	})
	s.event.On(event.INSTANCE_NEW, func(sessionId string, args ...interface{}) {
		instanceName := args[0].(string)
		log.Printf("EVENT: Instance New %s\n", instanceName)
		instance, err := s.storage.InstanceGet(instanceName)
		if err != nil {
			log.Printf("Instance [%s] was not found in storage. Got %s\n", instanceName, err)
			return
		}
		session, err := s.storage.SessionGet(instance.SessionId)
		if err != nil {
			log.Printf("Session [%s] was not found in storage. Got %s\n", instance.SessionId, err)
			return
		}
		s.scheduleInstance(instance, session.PlaygroundId)
	})
	s.event.On(event.INSTANCE_DELETE, func(sessionId string, args ...interface{}) {
		instanceName := args[0].(string)
		log.Printf("EVENT: Instance Delete %s\n", instanceName)
		instance := &types.Instance{Name: instanceName}
		s.unscheduleInstance(instance)
	})
	s.event.On(event.PLAYGROUND_NEW, func(playgroundId string, args ...interface{}) {
		s.mx.Lock()

		log.Printf("EVENT: Playground New %s\n", playgroundId)

		// Don't defer lock as updatePlaygrounds will lock again
		s.mx.Unlock()
		// We just update all playgrounds we manage to be safe. This is pretty fast anyway and this event should be fairly rare
		s.updatePlaygrounds()
	})
	s.started = true

	return nil
}
