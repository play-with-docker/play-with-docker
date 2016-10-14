package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/franela/play-with-docker/services"
	"github.com/go-zoo/bone"
)

func NewInstance(rw http.ResponseWriter, req *http.Request) {
	sessionId := bone.GetValue(req, "sessionId")

	s := services.GetSession(sessionId)

	s.Lock()
	if len(s.Instances) >= 5 {
		rw.WriteHeader(http.StatusConflict)
		return
	}

	i, err := services.NewInstance(s)
	s.Unlock()
	if err != nil {
		log.Println(err)
		//TODO: Set a status error
	} else {
		json.NewEncoder(rw).Encode(i)
	}
}
