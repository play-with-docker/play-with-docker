package handlers

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/play-with-docker/play-with-docker/config"
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

	if config.NoWindows {
		http.ServeFile(w, r, "./www/index-nw.html")
	} else {
		http.ServeFile(w, r, "./www/index.html")
	}
}
