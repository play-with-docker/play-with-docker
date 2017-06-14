package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/play-with-docker/play-with-docker/pwd"
)

func NewInstance(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionId := vars["sessionId"]

	body := pwd.InstanceConfig{Host: req.Host}

	json.NewDecoder(req.Body).Decode(&body)

	s := core.SessionGet(sessionId)

	if len(s.Instances) >= 5 {
		rw.WriteHeader(http.StatusConflict)
		return
	}

	i, err := core.InstanceNew(s, body)
	if err != nil {
		log.Println(err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
		//TODO: Set a status error
	} else {
		json.NewEncoder(rw).Encode(i)
	}
}
