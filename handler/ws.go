package handler

import (
	"fmt"
	"log"

	"github.com/franela/play-with-docker/services"
	"github.com/googollee/go-socket.io"
	"github.com/gorilla/mux"
)

func (h *handlers) ws(so socketio.Socket) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from ", r)
		}
	}()
	vars := mux.Vars(so.Request())

	sessionId := vars["sessionId"]

	session := services.GetSession(sessionId)
	if session == nil {
		log.Printf("Session with id [%s] does not exist!\n", sessionId)
		return
	}

	session.AddNewClient(services.NewClient(so, session))
}
func (h *handlers) wsError(so socketio.Socket) {
	log.Println("error ws")
}
