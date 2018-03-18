package handlers

import (
	"encoding/json"
	"log"
	"net/http"
<<<<<<< HEAD

	"github.com/play-with-docker/play-with-docker/services"
=======
>>>>>>> upstream/master
)

func GetInstanceImages(rw http.ResponseWriter, req *http.Request) {
	playground := core.PlaygroundFindByDomain(req.Host)
	if playground == nil {
		log.Printf("Playground for domain %s was not found!", req.Host)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	json.NewEncoder(rw).Encode(playground.AvailableDinDInstanceImages)
}
