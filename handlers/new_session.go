package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/franela/play-with-docker/services"
)

func NewSession(rw http.ResponseWriter, req *http.Request) {
	if !services.IsHuman(req) {
		// User it not a human
		rw.WriteHeader(http.StatusConflict)
		rw.Write([]byte("Only humans are allowed!"))
		return
	}

	s, err := services.NewSession()
	if err != nil {
		log.Println(err)
		//TODO: Return some error code
	} else {
		http.Redirect(rw, req, fmt.Sprintf("/p/%s", s.Id), http.StatusFound)
	}
}
