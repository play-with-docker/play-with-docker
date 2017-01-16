package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/franela/play-with-docker/core"
	"github.com/gorilla/mux"
)

func (h *handlers) newInstance(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionId := vars["sessionId"]

	s, err := h.core.GetSession(sessionId)
	if err != nil {
		if core.SessionNotFound(err) {
			rw.WriteHeader(http.StatusNotFound)
			return
		} else {
			log.Println(err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	i, err := h.core.NewInstance(s)

	if err != nil {
		if core.MaxInstancesInSessionReached(err) {
			rw.WriteHeader(http.StatusConflict)
			return
		} else {
			log.Println(err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	json.NewEncoder(rw).Encode(i)

	/*
		i, err := h.core(
		s.Lock()
		defer s.Unlock()
		if len(s.Instances) >= 5 {
			rw.WriteHeader(http.StatusConflict)
			return
		}

		i, err := services.NewInstance(s)
		if err != nil {
			log.Println(err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
			//TODO: Set a status error
		} else {
			json.NewEncoder(rw).Encode(i)
		}
	*/
}
