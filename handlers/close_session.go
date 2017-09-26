package handlers

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func CloseSession(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionId := vars["sessionId"]

	session := core.SessionGet(sessionId)
	if session == nil {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	if err := core.SessionClose(session); err != nil {
		log.Println(err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

}
