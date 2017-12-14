package handlers

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/play-with-docker/play-with-docker/storage"
)

func Home(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionId := vars["sessionId"]

	s, err := core.SessionGet(sessionId)
	if err == storage.NotFoundError {
		// Session doesn't exist (can happen if closing the sessions an reloading the page, or similar).
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if s.Stack != "" {
		go core.SessionDeployStack(s)
	}

	playground := core.PlaygroundGet(s.PlaygroundId)
	if playground == nil {
		log.Printf("Playground with id %s for session %s was not found!", s.PlaygroundId, s.Id)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	index := filepath.Join("./www", playground.AssetsDir, "/index.html")
	if _, err := os.Stat(index); os.IsNotExist(err) {
		index = "./www/default/index.html"
	}

	http.ServeFile(w, r, index)
}

func Landing(rw http.ResponseWriter, req *http.Request) {
	playground := core.PlaygroundFindByDomain(req.Host)
	if playground == nil {
		log.Printf("Playground for domain %s was not found!", req.Host)
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	rw.Write(landings[playground.Id])

}
