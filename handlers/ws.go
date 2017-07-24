package handlers

import (
	"fmt"
	"log"

	"github.com/googollee/go-socket.io"
	"github.com/gorilla/mux"
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

	client := core.ClientNew(so.Id(), session)

	so.On("session close", func() {
		core.SessionClose(session)
	})

	so.On("instance terminal in", func(name, data string) {
		// User wrote something on the terminal. Need to write it to the instance terminal
		core.InstanceWriteToTerminal(session.Id, name, data)
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
