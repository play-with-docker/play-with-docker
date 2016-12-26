package handlers

import (
	"crypto/tls"
	"encoding/json"
	"log"
	"net/http"

	"github.com/franela/play-with-docker/services"
	"github.com/gorilla/mux"
)

func SetKeys(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionId := vars["sessionId"]
	instanceName := vars["instanceName"]

	type certs struct {
		ServerCert []byte `json:"server_cert"`
		ServerKey  []byte `json:"server_key"`
	}

	var c certs
	jsonErr := json.NewDecoder(req.Body).Decode(&c)
	if jsonErr != nil {
		log.Println(jsonErr)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	s := services.GetSession(sessionId)
	s.Lock()
	defer s.Unlock()
	i := services.GetInstance(s, instanceName)

	i.ServerCert = c.ServerCert
	i.ServerKey = c.ServerKey

	cert, err := tls.X509KeyPair(i.ServerCert, i.ServerKey)
	if err != nil {
		log.Println(err)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	i.Cert = &cert
}
