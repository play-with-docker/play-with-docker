package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/franela/play-with-docker/services"
	"github.com/gorilla/mux"
)

func DeleteSession(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionId := vars["sessionId"]
	log.Println(sessionId)

	session := services.GetSession(sessionId)

	if session == nil {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	err := services.CloseSession(session)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	
	json.NewEncoder(rw).Encode(session)
}
