package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/franela/play-with-docker/core"
	"github.com/gorilla/mux"
)

func (h *handlers) setKeys(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionId := vars["sessionId"]
	//instanceName := vars["instanceName"]

	type certs struct {
		ServerCert []byte `json:"server_cert"`
		ServerKey  []byte `json:"server_key"`
	}

	var c certs
	jsonErr := json.NewDecoder(req.Body).Decode(&c)
	if jsonErr != nil {
		log.Println(jsonErr)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Printf("%#v", c)
	if len(c.ServerCert) == 0 || len(c.ServerKey) == 0 {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	_, err := h.core.GetSession(sessionId)
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
	/*
		s.Lock()
		defer s.Unlock()
		i := services.GetInstance(s, instanceName)

		_, err := i.SetCertificate(c.ServerCert, c.ServerKey)

		if err != nil {
			log.Println(err)
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		log.Printf("Set keys for instance %s\n", instanceName)
	*/
}
