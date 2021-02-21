package handlers

import (
	"io/fs"
	"log"
	"net/http"
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

	index, err := fs.ReadFile(staticFiles, filepath.Join(playground.AssetsDir, "/index.html"))
	if err != nil {
		index, err = fs.ReadFile(staticFiles, "default/index.html")
	}

	if err != nil {
		w.WriteHeader(http.StatusFound)
		return

	}
	w.Write(index)
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
