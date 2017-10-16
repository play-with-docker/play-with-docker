package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func GetUser(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	userId := vars["userId"]

	u, err := core.UserGet(userId)
	if err != nil {
		log.Println(err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(rw).Encode(u)
}
