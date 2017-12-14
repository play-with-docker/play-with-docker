package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/play-with-docker/play-with-docker/pwd"
)

func SessionSetup(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionId := vars["sessionId"]

	body := pwd.SessionSetupConf{PlaygroundFQDN: req.Host}

	json.NewDecoder(req.Body).Decode(&body)

	s, err := core.SessionGet(sessionId)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = core.SessionSetup(s, body)
	if err != nil {
		if pwd.SessionNotEmpty(err) {
			log.Println("Cannot setup a session that contains instances")
			rw.WriteHeader(http.StatusConflict)
			rw.Write([]byte("Cannot setup a session that contains instances"))
			return
		}
		log.Println(err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
}
