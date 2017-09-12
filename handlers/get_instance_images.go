package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/play-with-docker/play-with-docker/config"
)

func GetInstanceImages(rw http.ResponseWriter, req *http.Request) {
	instanceImages := []string{
		config.GetDindImageName(),
		"franela/dind:dev",
	}
	json.NewEncoder(rw).Encode(instanceImages)
}
