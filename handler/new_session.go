package handler

import (
	"fmt"
	"log"
	"net/http"
)

func (h *handlers) newSession(rw http.ResponseWriter, req *http.Request) {
	isHuman, err := h.recaptcha.IsHuman(req)
	if err != nil {
		log.Println(err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !isHuman {
		// User it not a human
		rw.WriteHeader(http.StatusConflict)
		rw.Write([]byte("Only humans are allowed!"))
		return
	}

	s, err := h.core.NewSession()
	if err != nil {
		log.Println(err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	} else {
		http.Redirect(rw, req, fmt.Sprintf("/p/%s", s.Id), http.StatusFound)
	}
}
