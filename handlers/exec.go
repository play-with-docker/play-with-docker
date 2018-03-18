package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type execRequest struct {
	Cmd []string `json:"command"`
}

type execResponse struct {
	ExitCode int `json:"status_code"`
}

func Exec(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionId := vars["sessionId"]
	instanceName := vars["instanceName"]

	var er execRequest
	err := json.NewDecoder(req.Body).Decode(&er)
	if err != nil {
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

	code, err := core.InstanceExec(i, er.Cmd)

	if err != nil {
		log.Println(err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(rw).Encode(execResponse{code})
}
