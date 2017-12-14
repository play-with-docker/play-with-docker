package handlers

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/play-with-docker/play-with-docker/storage"
)

func CloseSession(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionId := vars["sessionId"]

	session, err := core.SessionGet(sessionId)
	if err == storage.NotFoundError {
		rw.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := core.SessionClose(session); err != nil {
		log.Println(err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

}
