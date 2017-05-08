package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/play-with-docker/play-with-docker/services"
)

func NewInstance(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionId := vars["sessionId"]

	body := struct{ ImageName string }{}

	json.NewDecoder(req.Body).Decode(&body)

	s := services.GetSession(sessionId)

	s.Lock()
	defer s.Unlock()
	if len(s.Instances) >= 5 {
		rw.WriteHeader(http.StatusConflict)
		return
	}

	i, err := services.NewInstance(s, body.ImageName)
	if err != nil {
		log.Println(err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
		//TODO: Set a status error
	} else {
		json.NewEncoder(rw).Encode(i)
	}
}
