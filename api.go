package main

import (
	"log"
	"net/http"
	"os"

	"github.com/franela/play-with-docker/handlers"
	"github.com/franela/play-with-docker/services"
	"github.com/gorilla/mux"
	"github.com/urfave/negroni"
)

func main() {

	server := services.CreateWSServer()

	server.On("connection", handlers.WS)
	server.On("error", handlers.WSError)

	err := services.LoadSessionsFromDisk()
	if err != nil && !os.IsNotExist(err) {
		log.Fatal("Error decoding sessions from disk ", err)
	}

	r := mux.NewRouter()
	r.StrictSlash(false)

	r.HandleFunc("/ping", http.HandlerFunc(handlers.Ping)).Methods("GET")
	r.HandleFunc("/", http.HandlerFunc(handlers.NewSession)).Methods("GET")
	r.HandleFunc("/sessions/{sessionId}", http.HandlerFunc(handlers.GetSession)).Methods("GET")
	r.HandleFunc("/sessions/{sessionId}/instances", http.HandlerFunc(handlers.NewInstance)).Methods("POST")
	r.HandleFunc("/sessions/{sessionId}/instances/{instanceName}", http.HandlerFunc(handlers.DeleteInstance)).Methods("DELETE")

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./www/index.html")
	})
	r.HandleFunc("/p/{sessionId}", h).Methods("GET")
	r.PathPrefix("/assets").Handler(http.FileServer(http.Dir("./www")))

	r.Handle("/sessions/{sessionId}/ws/", server)

	n := negroni.Classic()
	n.UseHandler(r)

	log.Fatal(http.ListenAndServe("0.0.0.0:3000", n))

}
