package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/franela/play-with-docker/services"
	"github.com/gorilla/mux"
)

func GetSession(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionId := vars["sessionId"]

	session := services.GetSession(sessionId)

	if session == nil {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	json.NewEncoder(rw).Encode(session)
}
