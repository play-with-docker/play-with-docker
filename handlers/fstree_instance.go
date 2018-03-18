package handlers

import (
	"io"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func fsTree(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionId := vars["sessionId"]
	instanceName := vars["instanceName"]

	s, _ := core.SessionGet(sessionId)
	if s == nil {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	i := core.InstanceGet(s, instanceName)
	if i == nil {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	tree, err := core.InstanceFSTree(i)

	if err != nil {
		log.Println(err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	rw.Header().Set("content-type", "application/json")
	if _, err = io.Copy(rw, tree); err != nil {
		log.Println(err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
}
