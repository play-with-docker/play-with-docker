package handlers

import (
	"net/http"

	"github.com/franela/play-with-docker/services"
	"github.com/go-zoo/bone"
)

func DeleteInstance(rw http.ResponseWriter, req *http.Request) {
	sessionId := bone.GetValue(req, "sessionId")
	instanceId := bone.GetValue(req, "instanceId")

	s := services.GetSession(sessionId)
	i := services.GetInstance(s, instanceId)
	err := services.DeleteInstance(s, i)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
}
