package handler

import (
	"net/http"

	"github.com/franela/play-with-docker/core"
	"github.com/gorilla/mux"
)

func (h *handlers) deleteInstance(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionId := vars["sessionId"]
	instanceName := vars["instanceName"]

	err := h.core.DeleteInstance(sessionId, instanceName)
	if err != nil {
		if core.SessionNotFound(err) {
			rw.WriteHeader(http.StatusNotFound)
			return
		}
		if core.InstanceNotFound(err) {
			rw.WriteHeader(http.StatusNotFound)
			return
		}
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
}
