package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-zoo/bone"
	"github.com/xetorthio/play-with-docker/services"
)

func GetSession(rw http.ResponseWriter, req *http.Request) {
	sessionId := bone.GetValue(req, "sessionId")

	session := services.GetSession(sessionId)

	if session == nil {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	json.NewEncoder(rw).Encode(session)
}
