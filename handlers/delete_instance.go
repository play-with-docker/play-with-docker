package handlers

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/play-with-docker/play-with-docker/services"
)

func DeleteInstance(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionId := vars["sessionId"]
	instanceName := vars["instanceName"]

	s := services.GetSession(sessionId)
	s.Lock()
	defer s.Unlock()
	i := services.GetInstance(s, instanceName)
	err := services.DeleteInstance(s, i)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
}
