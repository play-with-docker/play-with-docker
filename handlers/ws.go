package handlers

import (
	"fmt"
	"log"

	"github.com/googollee/go-socket.io"
	"github.com/gorilla/mux"
	"github.com/play-with-docker/play-with-docker/services"
)

func WS(so socketio.Socket) {
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
func WSError(so socketio.Socket) {
	log.Println("error ws")
}

/*
	so.Join(sessionId)

	// TODO: Reset terminal geometry

	so.On("resize", func(cols, rows int) {
		// TODO: Reset terminal geometry
	})

	so.On("disconnection", func() {
		//TODO: reset the best terminal geometry
	})

	ctx := context.Background()

	session := services.GetSession(sessionId)
	instance := services.GetInstance(session, instanceName)

	if instance.Stdout == nil {
		id, err := services.CreateExecConnection(instance.Name, ctx)
		if err != nil {
			return
		}
		conn, err := services.AttachExecConnection(id, ctx)
		if err != nil {
			return
		}

		encoder := encoding.Replacement.NewEncoder()
		instance.Conn = conn
		instance.Stdout = &cookoo.MultiWriter{}
		instance.Stdout.Init()
		u1 := uuid.NewV4()
		instance.Stdout.AddWriter(u1.String(), ws)
		go func() {
			io.Copy(encoder.Writer(instance.Stdout), instance.Conn.Reader)
			instance.Stdout.RemoveWriter(u1.String())
		}()
		go func() {
			io.Copy(instance.Conn.Conn, ws)
			instance.Stdout.RemoveWriter(u1.String())
		}()
		select {
		case <-ctx.Done():
		}
	} else {
		u1 := uuid.NewV4()
		instance.Stdout.AddWriter(u1.String(), ws)

		go func() {
			io.Copy(instance.Conn.Conn, ws)
			instance.Stdout.RemoveWriter(u1.String())
		}()
		select {
		case <-ctx.Done():
		}
	}
}
*/
