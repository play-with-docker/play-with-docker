package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path"

	"github.com/play-with-docker/play-with-docker/config"
	"github.com/play-with-docker/play-with-docker/services"
)

type NewSessionResponse struct {
	SessionId string `json:"session_id"`
	Hostname  string `json:"hostname"`
}

func NewSession(rw http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	if !services.IsHuman(req, rw) {
		// User it not a human
		rw.WriteHeader(http.StatusForbidden)
		return
	}

	reqDur := req.Form.Get("session-duration")
	stack := req.Form.Get("stack")

	duration := services.GetDuration(reqDur)
	s, err := services.NewSession(duration)
	if err != nil {
		log.Println(err)
		//TODO: Return some error code
	} else {
		s.StackFile = stack
		if stack != "" {
			go deployStack(s)
		}
		hostname := fmt.Sprintf("%s.%s", config.PWDCName, req.Host)
		// If request is not a form, return sessionId in the body
		if req.Header.Get("X-Requested-With") == "XMLHttpRequest" {
			resp := NewSessionResponse{SessionId: s.Id, Hostname: hostname}
			rw.Header().Set("Content-Type", "application/json")
			json.NewEncoder(rw).Encode(resp)
			return
		}
		http.Redirect(rw, req, fmt.Sprintf("http://%s/p/%s", hostname, s.Id), http.StatusFound)
	}
}

func deployStack(s *services.Session) {
	i, err := services.NewInstance(s, services.InstanceConfig{})
	if err != nil {
		log.Printf("Error creating instance for stack [%s]: %s\n", s.StackFile, err)
	}
	err = i.UploadFromURL("https://raw.githubusercontent.com/play-with-docker/stacks/master" + s.StackFile)
	if err != nil {
		log.Printf("Error uploading stack file [%s]: %s\n", s.StackFile, err)
	}

	fileName := path.Base(s.StackFile)
	code, err := services.Exec(i.Name, []string{"docker-compose", "-f", "/var/run/pwd/uploads/" + fileName, "up", "-d"})
	if err != nil {
		log.Printf("Error executing stack [%s]: %s\n", s.StackFile, err)
	}

	log.Printf("Stack execution finished with code %d\n", code)
}
