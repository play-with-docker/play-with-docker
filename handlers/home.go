package handlers

import (
	"net/http"

	"github.com/gorilla/mux"
)

func Home(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionId := vars["sessionId"]

	s := core.SessionGet(sessionId)
	if s.Stack != "" {
		go core.SessionDeployStack(s)
	}
	http.ServeFile(w, r, "./www/index.html")
}
