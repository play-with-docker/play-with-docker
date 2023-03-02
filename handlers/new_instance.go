package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/thebsdbox/play-with-docker/provisioner"
	"github.com/thebsdbox/play-with-docker/pwd/types"
)

func NewInstance(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionId := vars["sessionId"]

	body := types.InstanceConfig{PlaygroundFQDN: req.Host, DindVolumeSize: "5G"}

	json.NewDecoder(req.Body).Decode(&body)

	s, err := core.SessionGet(sessionId)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	playground := core.PlaygroundGet(s.PlaygroundId)
	if playground == nil {
		log.Printf("Playground with id %s for session %s was not found!", s.PlaygroundId, s.Id)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	if body.Type == "windows" && !playground.AllowWindowsInstances {
		rw.WriteHeader(http.StatusUnauthorized)
		return
	}

	instances, err := core.InstanceFindBySession(s)

	if err != nil {
		log.Println(err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	if playground.MaxInstances > 0 && len(instances) >= playground.MaxInstances {
		log.Println(err)
		rw.WriteHeader(http.StatusConflict)
		return
	}

	if len(playground.DindVolumeSize) > 0 {
		body.DindVolumeSize = playground.DindVolumeSize
	}

	// TODO I don't like how this is implemented here. NewInstance
	// should be a function that's in the Playground struct.
	if playground.Privileged {
		body.Privileged = true
	}

	i, err := core.InstanceNew(s, body)
	if err != nil {
		if provisioner.OutOfCapacity(err) {
			rw.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintln(rw, `{"error": "out_of_capacity"}`)
			return
		}
		log.Println(err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
		//TODO: Set a status error
	} else {
		json.NewEncoder(rw).Encode(i)
	}
}
