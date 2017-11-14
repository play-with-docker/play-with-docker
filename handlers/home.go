package handlers

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func Home(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionId := vars["sessionId"]

	s := core.SessionGet(sessionId)
	if s == nil {
		// Session doesn't exist (can happen if closing the sessions an reloading the page, or similar).
		w.WriteHeader(http.StatusNotFound)
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

	if !playground.AllowWindowsInstances {
		http.ServeFile(w, r, "./www/index-nw.html")
	} else {
		http.ServeFile(w, r, "./www/index.html")
	}
}
