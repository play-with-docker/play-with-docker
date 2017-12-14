package handlers

import (
	"io"
	"log"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/play-with-docker/play-with-docker/storage"
)

func FileUpload(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessionId := vars["sessionId"]
	instanceName := vars["instanceName"]

	s, err := core.SessionGet(sessionId)
	if err == storage.NotFoundError {
		rw.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	i := core.InstanceGet(s, instanceName)

	// allow up to 32 MB which is the default

	// has a url query parameter, ignore body
	if url := req.URL.Query().Get("url"); url != "" {

		_, fileName := filepath.Split(url)

		err := core.InstanceUploadFromUrl(i, fileName, "", req.URL.Query().Get("url"))
		if err != nil {
			log.Println(err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		rw.WriteHeader(http.StatusOK)
		return
	} else {
		red, err := req.MultipartReader()
		if err != nil {
			log.Println(err)
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		path := req.URL.Query().Get("path")

		for {
			p, err := red.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Println(err)
				continue
			}

			if p.FileName() == "" {
				continue
			}
			err = core.InstanceUploadFromReader(i, p.FileName(), path, p)
			if err != nil {
				log.Println(err)
				rw.WriteHeader(http.StatusInternalServerError)
				return
			}

			log.Printf("Uploaded [%s] to [%s]\n", p.FileName(), i.Name)
		}
		rw.WriteHeader(http.StatusOK)
		return
	}

}
