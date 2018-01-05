package handlers

import (
	"encoding/base64"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func file(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionId := vars["sessionId"]
	instanceName := vars["instanceName"]

	query := req.URL.Query()

	path := query.Get("path")

	if path == "" {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

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

	instanceFile, err := core.InstanceFile(i, path)

	if err != nil {
		log.Println(err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	encoder := base64.NewEncoder(base64.StdEncoding, rw)

	if _, err = io.Copy(encoder, instanceFile); err != nil {
		log.Println(err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
}
