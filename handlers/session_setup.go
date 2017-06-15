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

	body := pwd.SessionSetupConf{}

	json.NewDecoder(req.Body).Decode(&body)

	s := core.SessionGet(sessionId)

	if len(s.Instances) > 0 {
		log.Println("Cannot setup a session that contains instances")
		rw.WriteHeader(http.StatusConflict)
		rw.Write([]byte("Cannot setup a session that contains instances"))
		return
	}

	s.Host = req.Host
	err := core.SessionSetup(s, body)
	if err != nil {
		log.Println(err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
}
