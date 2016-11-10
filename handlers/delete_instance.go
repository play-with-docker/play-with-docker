package handlers

import (
	"net/http"

	"github.com/franela/play-with-docker/services"
	"github.com/gorilla/mux"
)

func DeleteInstance(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionId := vars["sessionId"]
	instanceName := vars["instanceName"]

	s := services.GetSession(sessionId)
	i := services.GetInstance(s, instanceName)
	err := services.DeleteInstance(s, i)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
}
