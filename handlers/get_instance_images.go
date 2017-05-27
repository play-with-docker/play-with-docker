package handlers

import (
	"encoding/json"
	"net/http"
)

func GetInstanceImages(rw http.ResponseWriter, req *http.Request) {
	instanceImages := core.InstanceAllowedImages()
	json.NewEncoder(rw).Encode(instanceImages)
}
