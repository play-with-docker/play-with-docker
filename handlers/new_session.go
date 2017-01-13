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
		// If request is not a form, return sessionId in the body
		if req.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			rw.Write([]byte(s.Id))
			return
		}
		http.Redirect(rw, req, fmt.Sprintf("/p/%s", s.Id), http.StatusFound)
	}
}
