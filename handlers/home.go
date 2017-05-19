package handlers

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/play-with-docker/play-with-docker/services"
)

func Home(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionId := vars["sessionId"]

	s := services.GetSession(sessionId)
	if s == nil {
		// Session doesn't exist (can happen if closing the sessions an reloading the page, or similar).
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if s.Stack != "" {
		go s.DeployStack()
	}
	http.ServeFile(w, r, "./www/index.html")
}
