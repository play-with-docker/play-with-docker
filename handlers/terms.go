package handlers

import (
	"log"
	"net"
	"sync"

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
type manager struct {
	sendCh    chan info
	receiveCh chan info
	terminals map[string]terminal
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
func (m *manager) connect(instance *types.Instance) error {
	if !m.trackingInstance(instance) {
		return nil
	}

	conn, err := core.InstanceGetTerminal(instance)
	if err != nil {
		return err
	}
	chw := make(chan []byte, 10)
	t := terminal{conn: conn, write: chw, instance: instance}
	m.terminals[instance.Name] = t
	t.Go(m.receiveCh, m.errorCh)
	return nil
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

	t := m.terminals[instance.Name]
	if t.write != nil {
		close(t.write)
	}
	if t.conn != nil {
		t.conn.Close()
	}
	m.untrackInstance(instance)
}

func (m *manager) process() {
	for {
		select {
		case i := <-m.sendCh:
			t := m.terminals[i.name]
			t.write <- i.data
		case instance := <-m.errorCh:
			// check if it still exists before reconnecting
			i := core.InstanceGet(&types.Session{Id: instance.SessionId}, instance.Name)
			if i == nil {
				log.Println("Instance doest not exist anymore. Won't reconnect")
				continue
			}
			log.Println("reconnecting")
			m.connect(instance)
		}
	}
}
func (m *manager) Close() {
	for _, i := range m.instances {
		m.disconnect(i)
	}
}

func NewManager(s *types.Session) (*manager, error) {
	m := &manager{
		sendCh:    make(chan info),
		receiveCh: make(chan info),
		terminals: make(map[string]terminal),
		errorCh:   make(chan *types.Instance),
		instances: make(map[string]*types.Instance),
	}

	instances, err := core.InstanceFindBySession(s)
	if err != nil {
		return nil, err
	}
	for _, i := range instances {
		m.instances[i.Name] = i
		m.connect(i)
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

	go m.process()
	return m, nil
}
