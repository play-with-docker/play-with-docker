package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/play-with-docker/play-with-docker/event"
	"github.com/play-with-docker/play-with-docker/storage"
	"github.com/satori/go.uuid"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type message struct {
	Name string        `json:"name"`
	Args []interface{} `json:"args"`
}

type socket struct {
	c         *websocket.Conn
	mx        sync.Mutex
	listeners map[string][]func(args ...interface{})
	r         *http.Request
	id        string
	closed    bool
}

func newSocket(r *http.Request, c *websocket.Conn) *socket {
	return &socket{
		c:         c,
		listeners: map[string][]func(args ...interface{}){},
		r:         r,
		id:        uuid.NewV4().String(),
	}
}

func (s *socket) Id() string {
	return s.id
}

func (s *socket) Request() *http.Request {
	return s.r
}

func (s *socket) Close() {
	s.closed = true
	s.onMessage(message{Name: "close"})
}

func (s *socket) process() {
	defer s.Close()
	for {
		mt, m, err := s.c.ReadMessage()
		if err != nil {
			log.Printf("Error reading message from websocket. Got: %v\n", err)
			break
		}
		if mt != websocket.TextMessage {
			log.Printf("Received websocket message, but it is not a text message.\n")
			continue
		}
		go func() {
			var msg message
			if err := json.Unmarshal(m, &msg); err != nil {
				log.Printf("Cannot unmarshal message received from websocket. Got: %v\n", err)
				return
			}
			s.onMessage(msg)
		}()
	}
}

func (s *socket) onMessage(msg message) {
	s.mx.Lock()
	defer s.mx.Unlock()

	cbs, found := s.listeners[msg.Name]
	if !found {
		return
	}
	for _, cb := range cbs {
		go cb(msg.Args...)
	}
}

func (s *socket) Emit(ev string, args ...interface{}) {
	s.mx.Lock()
	defer s.mx.Unlock()

	if s.closed {
		return
	}

	m := message{Name: ev, Args: args}
	b, err := json.Marshal(m)
	if err != nil {
		log.Printf("Cannot marshal event to json. Got: %v\n", err)
		return
	}
	if err := s.c.WriteMessage(websocket.TextMessage, b); err != nil {
		log.Printf("Cannot write event to websocket connection. Got: %v\n", err)
		s.Close()
		return
	}
}

func (s *socket) On(ev string, cb func(args ...interface{})) {
	s.mx.Lock()
	defer s.mx.Unlock()
	listeners, found := s.listeners[ev]
	if !found {
		listeners = []func(args ...interface{}){}
	}
	listeners = append(listeners, cb)
	s.listeners[ev] = listeners
}

func WSH(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()

	s := newSocket(r, c)
	ws(s)
	s.process()
}

func ws(so *socket) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from ", r)
		}
	}()
	vars := mux.Vars(so.Request())

	sessionId := vars["sessionId"]

	session, err := core.SessionGet(sessionId)
	if err == storage.NotFoundError {
		log.Printf("Session with id [%s] does not exist!\n", sessionId)
		return
	}

	client := core.ClientNew(so.Id(), session)
	if client == nil {
		log.Printf("ERROR: Client was not created for session id %s and socket id %s\n", session.Id, so.Id())
	}

	m, err := NewManager(session)
	if err != nil {
		log.Printf("Error creating terminal manager. Got: %v", err)
		return
	}

	go m.Receive(func(name string, data []byte) {
		so.Emit("instance terminal out", name, string(data))
	})
	go m.Status(func(name, status string) {
		so.Emit("instance terminal status", name, status)
	})

	err = m.Start()
	if err != nil {
		log.Println(err)
		return
	}

	so.On("session close", func(args ...interface{}) {
		m.Close()
		core.SessionClose(session)
	})

	so.On("instance terminal in", func(args ...interface{}) {
		if len(args) == 2 && args[0] != nil && args[1] != nil {
			name := args[0].(string)
			data := args[1].(string)
			m.Send(name, []byte(data))
		}
	})

	so.On("instance viewport resize", func(args ...interface{}) {
		if len(args) == 2 && args[0] != nil && args[1] != nil {
			// User resized his viewport
			cols := args[0].(float64)
			rows := args[1].(float64)
			core.ClientResizeViewPort(client, uint(cols), uint(rows))
		}
	})

	so.On("close", func(args ...interface{}) {
		m.Close()
		core.ClientClose(client)
	})

	e.OnAny(func(eventType event.EventType, sessionId string, args ...interface{}) {
		if session.Id == sessionId {
			so.Emit(eventType.String(), args...)
		}
	})
}
