package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/play-with-docker/play-with-docker/services"
)

func GetInstanceImages(rw http.ResponseWriter, req *http.Request) {
	instanceImages := services.InstanceImages()
	json.NewEncoder(rw).Encode(instanceImages)
}
