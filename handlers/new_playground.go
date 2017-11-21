package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/play-with-docker/play-with-docker/config"
	"github.com/play-with-docker/play-with-docker/pwd/types"
)

func NewPlayground(rw http.ResponseWriter, req *http.Request) {
	if !validateToken(req) {
		rw.WriteHeader(http.StatusForbidden)
		return
	}

	var playground types.Playground

	err := json.NewDecoder(req.Body).Decode(&playground)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(rw, "Error creating playground. Got: %v", err)
		return
	}

	newPlayground, err := core.PlaygroundNew(playground)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(rw, "Error creating playground. Got: %v", err)
		return
	}

	json.NewEncoder(rw).Encode(newPlayground)
}

func ListPlaygrounds(rw http.ResponseWriter, req *http.Request) {
	if !validateToken(req) {
		rw.WriteHeader(http.StatusForbidden)
		return
	}

	playgrounds, err := core.PlaygroundList()
	if err != nil {
		log.Printf("Error listing playgrounds. Got: %v\n", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(rw).Encode(playgrounds)
}

func validateToken(req *http.Request) bool {
	_, password, ok := req.BasicAuth()
	if !ok {
		return false
	}

	if password != config.AdminToken {
		return false
	}

	return true
}
