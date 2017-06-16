package handlers

import (
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
		s.Host = r.Host
		go core.SessionDeployStack(s)
	}

	http.ServeFile(w, r, "./www/index.html")
}
