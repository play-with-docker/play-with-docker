package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/franela/play-with-docker/config"
	"github.com/franela/play-with-docker/services"
)

func NewSession(rw http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	if !services.IsHuman(req) {
		// User it not a human
		rw.WriteHeader(http.StatusConflict)
		rw.Write([]byte("Only humans are allowed!"))
		return
	}

	reqDur := req.Form.Get("session-duration")

	duration := services.GetDuration(reqDur)
	s, err := services.NewSession(duration)
	if err != nil {
		log.Println(err)
		//TODO: Return some error code
	} else {
		// If request is not a form, return sessionId in the body
		if req.Header.Get("X-Requested-With") == "XMLHttpRequest" {
			rw.Write([]byte(s.Id))
			return
		}
		http.Redirect(rw, req, fmt.Sprintf("http://%s.%s/p/%s", config.PWDCName, req.Host, s.Id), http.StatusFound)
	}
}
