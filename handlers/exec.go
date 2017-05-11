package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/play-with-docker/play-with-docker/services"
)

type execRequest struct {
	Cmd []string `json:"command"`
}

type execResponse struct {
	ExitCode int `json:"status_code"`
}

func Exec(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	instanceName := vars["instanceName"]

	var er execRequest
	err := json.NewDecoder(req.Body).Decode(&er)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	code, err := services.Exec(instanceName, er.Cmd)

	if err != nil {
		log.Println(err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(rw).Encode(execResponse{code})
}
