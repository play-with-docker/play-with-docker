package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/play-with-docker/play-with-docker/pwd/types"
	"github.com/play-with-docker/play-with-docker/storage"
)

type SessionInfo struct {
	*types.Session
	Instances map[string]*types.Instance `json:"instances"`
}

func GetSession(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionId := vars["sessionId"]

	session, err := core.SessionGet(sessionId)
	if err == storage.NotFoundError {
		rw.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	instances, err := core.InstanceFindBySession(session)
	if err != nil {
		log.Println(err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	is := map[string]*types.Instance{}
	for _, i := range instances {
		is[i.Name] = i
	}

	json.NewEncoder(rw).Encode(SessionInfo{session, is})
}
