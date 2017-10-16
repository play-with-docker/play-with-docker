package handlers

import (
	"fmt"
	"log"

	"github.com/googollee/go-socket.io"
	"github.com/gorilla/mux"
	"github.com/play-with-docker/play-with-docker/event"
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

	client := core.ClientNew(so.Id(), session)

	so.Join(session.Id)

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

	so.On("session close", func() {
		m.Close()
		core.SessionClose(session)
	})

	so.On("instance terminal in", func(name, data string) {
		m.Send(name, []byte(data))
	})

	so.On("instance viewport resize", func(cols, rows uint) {
		// User resized his viewport
		core.ClientResizeViewPort(client, cols, rows)
	})

	so.On("disconnection", func() {
		m.Close()
		core.ClientClose(client)
	})

	so.On("session keep alive", func() {
		e.Emit(event.SESSION_KEEP_ALIVE, sessionId)
	})
}

func WSError(so socketio.Socket) {
	log.Println("error ws")
}
