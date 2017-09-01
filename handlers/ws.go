package handlers

import (
	"fmt"
	"log"
	"net"
	"sync"

	"golang.org/x/text/encoding"

	"github.com/googollee/go-socket.io"
	"github.com/gorilla/mux"
	"github.com/play-with-docker/play-with-docker/event"
	"github.com/play-with-docker/play-with-docker/pwd/types"
)

func WS(so socketio.Socket) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from ", r)
		}
	}()
	vars := mux.Vars(so.Request())

	sessionId := vars["sessionId"]

	session := core.SessionGet(sessionId)
	if session == nil {
		log.Printf("Session with id [%s] does not exist!\n", sessionId)
		return
	}

	so.Join(session.Id)

	instances, err := core.InstanceFindBySession(session)
	if err != nil {
		log.Printf("Couldn't find instances for session with id [%s]. Got: %v\n", sessionId, err)
		return
	}
	var rw sync.Mutex
	trackedTerminals := make(map[string]net.Conn, len(instances))

	attachTerminalToSocket := func(instance *types.Instance, ws socketio.Socket) {
		rw.Lock()
		defer rw.Unlock()
		if _, found := trackedTerminals[instance.Name]; found {
			return
		}
		conn, err := core.InstanceGetTerminal(instance)
		if err != nil {
			log.Println(err)
			return
		}
		trackedTerminals[instance.Name] = conn

		go func(instanceName string, c net.Conn, ws socketio.Socket) {
			defer c.Close()
			defer func() {
				rw.Lock()
				defer rw.Unlock()
				delete(trackedTerminals, instanceName)
			}()
			encoder := encoding.Replacement.NewEncoder()
			buf := make([]byte, 1024)
			for {
				n, err := c.Read(buf)
				if err != nil {
					log.Println(err)
					return
				}
				b, err := encoder.Bytes(buf[:n])
				if err != nil {
					log.Println(err)
					return
				}
				ws.Emit("instance terminal out", instanceName, string(b))
			}
		}(instance.Name, conn, ws)
	}
	// since this is a new connection, get all terminals of the session and attach
	for _, instance := range instances {
		attachTerminalToSocket(instance, so)
	}

	e.On(event.INSTANCE_NEW, func(sessionId string, args ...interface{}) {
		if sessionId != session.Id {
			return
		}

		// There is a new instance in a session we are tracking. We should track it's terminal
		instanceName := args[0].(string)
		instance := core.InstanceGet(session, instanceName)
		if instance == nil {
			log.Printf("Instance [%s] was not found in session [%s]\n", instanceName, sessionId)
			return
		}
		attachTerminalToSocket(instance, so)
	})

	client := core.ClientNew(so.Id(), session)

	so.On("session close", func() {
		core.SessionClose(session)
	})

	so.On("instance terminal in", func(name, data string) {
		rw.Lock()
		defer rw.Unlock()
		conn, found := trackedTerminals[name]
		if !found {
			log.Printf("Could not find instance [%s] in session [%s]\n", name, sessionId)
			return
		}
		go conn.Write([]byte(data))
	})

	so.On("instance viewport resize", func(cols, rows uint) {
		// User resized his viewport
		core.ClientResizeViewPort(client, cols, rows)
	})

	so.On("disconnection", func() {
		core.ClientClose(client)
	})
}

func WSError(so socketio.Socket) {
	log.Println("error ws")
}
