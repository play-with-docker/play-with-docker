package handler

import (
	"encoding/json"
	"net/http"

	"github.com/franela/play-with-docker/core"
	"github.com/gorilla/mux"
)

func (h *handlers) getSession(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionId := vars["sessionId"]

	session, err := h.core.GetSession(sessionId)

	if err != nil {
		if core.SessionNotFound(err) {
			rw.WriteHeader(http.StatusNotFound)
			return
		} else {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	json.NewEncoder(rw).Encode(session)
}
