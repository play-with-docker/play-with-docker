package handlers

import (
	"net/http"

	"github.com/franela/play-with-docker/services"
	"github.com/go-zoo/bone"
)

func DeleteInstance(rw http.ResponseWriter, req *http.Request) {
	sessionId := bone.GetValue(req, "sessionId")
	instanceName := bone.GetValue(req, "instanceName")

	s := services.GetSession(sessionId)
	i := services.GetInstance(s, instanceName)
	err := services.DeleteInstance(s, i)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
}
