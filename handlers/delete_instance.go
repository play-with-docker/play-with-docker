package handlers

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/play-with-docker/play-with-docker/storage"
)

func DeleteInstance(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionId := vars["sessionId"]
	instanceName := vars["instanceName"]

	s, err := core.SessionGet(sessionId)
	if s != nil {
		i := core.InstanceGet(s, instanceName)
		err := core.InstanceDelete(s, i)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else if err == storage.NotFoundError {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	} else if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
}
