package handlers

import (
	"log"
	"net"
	"sync"
	"time"

	"github.com/play-with-docker/play-with-docker/event"
	"github.com/play-with-docker/play-with-docker/pwd/types"

	"golang.org/x/text/encoding"
)

type terminal struct {
	conn     net.Conn
	write    chan []byte
	instance *types.Instance
}

func (t *terminal) Go(ch chan info, ech chan *types.Instance) {
	go func() {
		for d := range t.write {
			_, err := t.conn.Write(d)
			if err != nil {
				ech <- t.instance
				return
			}
		}
	}()
	go func() {
		encoder := encoding.Replacement.NewEncoder()
		buf := make([]byte, 1024)
		for {
			n, err := t.conn.Read(buf)
			if err != nil {
				ech <- t.instance
				return
			}
			b, err := encoder.Bytes(buf[:n])
			if err != nil {
				ech <- t.instance
				return
			}
			ch <- info{name: t.instance.Name, data: b}
		}
	}()
}

type info struct {
	name string
	data []byte
}

type state struct {
	name   string
	status string
}

type manager struct {
	session   *types.Session
	sendCh    chan info
	receiveCh chan info
	stateCh   chan state
	terminals map[string]*terminal
	errorCh   chan *types.Instance
	instances map[string]*types.Instance
	sync.Mutex
}

func (m *manager) Send(name string, data []byte) {
	m.sendCh <- info{name: name, data: data}
}
func (m *manager) Receive(cb func(name string, data []byte)) {
	for i := range m.receiveCh {
		cb(i.name, i.data)
	}
}
func (m *manager) Status(cb func(name, status string)) {
	for s := range m.stateCh {
		cb(s.name, s.status)
	}
}

func (m *manager) connect(instance *types.Instance) error {
	if !m.trackingInstance(instance) {
		return nil
	}

	return m.connectTerminal(instance)
}

func (m *manager) connectTerminal(instance *types.Instance) error {
	m.Lock()
	defer m.Unlock()

	conn, err := core.InstanceGetTerminal(instance)
	if err != nil {
		return err
	}
	chw := make(chan []byte, 10)
	t := terminal{conn: conn, write: chw, instance: instance}
	m.terminals[instance.Name] = &t
	t.Go(m.receiveCh, m.errorCh)
	m.stateCh <- state{name: instance.Name, status: "connect"}

	return nil
}

func (m *manager) disconnectTerminal(instance *types.Instance) {
	m.Lock()
	defer m.Unlock()

	t := m.terminals[instance.Name]
	if t != nil {
		if t.write != nil {
			close(t.write)
		}
		if t.conn != nil {
			t.conn.Close()
		}
		delete(m.terminals, instance.Name)
	}
}

func (m *manager) getTerminal(instanceName string) *terminal {
	return m.terminals[instanceName]
}

func (m *manager) trackInstance(instance *types.Instance) {
	m.Lock()
	defer m.Unlock()

	m.instances[instance.Name] = instance

}
func (m *manager) untrackInstance(instance *types.Instance) {
	m.Lock()
	defer m.Unlock()

	delete(m.instances, instance.Name)
}
func (m *manager) trackingInstance(instance *types.Instance) bool {
	m.Lock()
	defer m.Unlock()
	_, found := m.instances[instance.Name]

	return found
}

func (m *manager) disconnect(instance *types.Instance) {
	if !m.trackingInstance(instance) {
		return
	}

	m.disconnectTerminal(instance)
	m.untrackInstance(instance)
}

func (m *manager) process() {
	for {
		select {
		case i := <-m.sendCh:
			t := m.getTerminal(i.name)
			if t != nil {
				t.write <- i.data
			}
		case instance := <-m.errorCh:
			// check if it still exists before reconnecting
			i := core.InstanceGet(&types.Session{Id: instance.SessionId}, instance.Name)
			if i == nil {
				log.Println("Instance doesn't exist anymore. Won't reconnect")
				continue
			}
			m.stateCh <- state{name: instance.Name, status: "reconnect"}
			time.AfterFunc(time.Second, func() {
				m.connect(instance)
			})
		}
	}
}
func (m *manager) Close() {
	for _, i := range m.instances {
		m.disconnect(i)
	}
}

func (m *manager) Start() error {
	instances, err := core.InstanceFindBySession(m.session)
	if err != nil {
		return err
	}
	for _, i := range instances {
		m.instances[i.Name] = i
		m.connect(i)
	}
	go m.process()
	return nil
}

func NewManager(s *types.Session) (*manager, error) {
	m := &manager{
		session:   s,
		sendCh:    make(chan info, 10),
		receiveCh: make(chan info, 10),
		stateCh:   make(chan state, 10),
		terminals: make(map[string]*terminal),
		errorCh:   make(chan *types.Instance, 10),
		instances: make(map[string]*types.Instance),
	}

	e.On(event.INSTANCE_NEW, func(sessionId string, args ...interface{}) {
		if sessionId != s.Id {
			return
		}

		// There is a new instance in a session we are tracking. We should track it's terminal
		instanceName := args[0].(string)
		instance := core.InstanceGet(s, instanceName)
		if instance == nil {
			log.Printf("Instance [%s] was not found in session [%s]\n", instanceName, sessionId)
			return
		}
		m.trackInstance(instance)
		m.connect(instance)
	})

	e.On(event.INSTANCE_DELETE, func(sessionId string, args ...interface{}) {
		if sessionId != s.Id {
			return
		}

		// There is a new instance in a session we are tracking. We should track it's terminal
		instanceName := args[0].(string)
		instance := &types.Instance{Name: instanceName}
		m.disconnect(instance)
	})

	return m, nil
}
