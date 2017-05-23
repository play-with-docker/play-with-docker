package handlers

import (
	"net/http"

	"github.com/gorilla/mux"
)

func DeleteInstance(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionId := vars["sessionId"]
	instanceName := vars["instanceName"]

	s := core.SessionGet(sessionId)
	i := core.InstanceGet(s, instanceName)
	err := core.InstanceDelete(s, i)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
}
