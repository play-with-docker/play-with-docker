package handlers

import "net/http"

func GetKeys(rw http.ResponseWriter, req *http.Request) {
	http.ServeFile(rw, req, "./pwd/keys.tar")
}
