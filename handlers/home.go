package handlers

import (
	"log"
	"net/http"
	"path"

	"github.com/gorilla/mux"
	"github.com/play-with-docker/play-with-docker/services"
)

func Home(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionId := vars["sessionId"]

	stack := r.URL.Query().Get("stack")
	s := services.GetSession(sessionId)
	if stack != "" {
		go deployStack(s, stack)
	}
	http.ServeFile(w, r, "./www/index.html")
}

func deployStack(s *services.Session, stack string) {
	i, err := services.NewInstance(s, services.InstanceConfig{})
	if err != nil {
		log.Printf("Error creating instance for stack [%s]: %s\n", stack, err)
	}
	err = i.UploadFromURL("https://raw.githubusercontent.com/play-with-docker/stacks/master" + stack)
	if err != nil {
		log.Printf("Error uploading stack file [%s]: %s\n", stack, err)
	}

	fileName := path.Base(stack)
	code, err := services.Exec(i.Name, []string{"docker-compose", "-f", "/var/run/pwd/uploads/" + fileName, "up", "-d"})
	if err != nil {
		log.Printf("Error executing stack [%s]: %s\n", stack, err)
	}

	log.Printf("Stack execution finished with code %d\n", code)
}
