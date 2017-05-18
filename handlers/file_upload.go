package handlers

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/play-with-docker/play-with-docker/services"
)

func FileUpload(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionId := vars["sessionId"]
	instanceName := vars["instanceName"]

	s := services.GetSession(sessionId)
	i := services.GetInstance(s, instanceName)

	// allow up to 32 MB which is the default

	// has a url query parameter, ignore body
	if url := req.URL.Query().Get("url"); url != "" {
		err := i.UploadFromURL(req.URL.Query().Get("url"))
		if err != nil {
			log.Println(err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		rw.WriteHeader(http.StatusOK)
		return
	} else {
		// This is for multipart upload
		log.Println("Not implemented yet")

		/*
			err := req.ParseMultipartForm(32 << 20)
			if err != nil {
				log.Println(err)
				rw.WriteHeader(http.StatusBadRequest)
				return
			}
		*/
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

}
