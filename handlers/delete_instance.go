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
	if s != nil {
		i := core.InstanceGet(s, instanceName)
		err := core.InstanceDelete(s, i)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else {
		rw.WriteHeader(http.StatusNotFound)
		return
	}
}
