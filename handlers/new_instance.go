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
	i, err := services.NewInstance(s)
	if err != nil {
		log.Println(err)
		//TODO: Set a status error
	} else {
		json.NewEncoder(rw).Encode(i)
	}
}
