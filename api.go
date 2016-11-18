package main

import (
	"log"
	"net/http"
	"os"

	"github.com/franela/play-with-docker/handlers"
	"github.com/franela/play-with-docker/services"
	"github.com/franela/play-with-docker/templates"
	"github.com/gorilla/mux"
	"github.com/urfave/negroni"
	"flag"
	"strconv"
)

func main() {

	var portNumber int
	flag.IntVar(&portNumber, "port", 3000, "Give a TCP port to run the application")
	flag.Parse()

	welcome, tmplErr := templates.GetWelcomeTemplate()
	if tmplErr != nil {
		log.Fatal(tmplErr)
	}

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
	r.HandleFunc("/", http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Write(welcome)
	})).Methods("GET")
	r.HandleFunc("/", http.HandlerFunc(handlers.NewSession)).Methods("POST")

	r.HandleFunc("/sessions/{sessionId}", http.HandlerFunc(handlers.GetSession)).Methods("GET")
	r.HandleFunc("/sessions/{sessionId}/instances", http.HandlerFunc(handlers.NewInstance)).Methods("POST")
	r.HandleFunc("/sessions/{sessionId}/instances/{instanceName}", http.HandlerFunc(handlers.DeleteInstance)).Methods("DELETE")

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./www/index.html")
	})
	r.HandleFunc("/p/{sessionId}", h).Methods("GET")
	r.PathPrefix("/assets").Handler(http.FileServer(http.Dir("./www")))
	r.HandleFunc("/robots.txt", http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		http.ServeFile(rw, r, "www/robots.txt")
	}))

	r.Handle("/sessions/{sessionId}/ws/", server)

	// Reverse proxy
	r.Host(`{node}-{port:[0-9]*}.play-with-docker.com`).Handler(handlers.NewMultipleHostReverseProxy())
	r.Host(`{node}.play-with-docker.com`).Handler(handlers.NewMultipleHostReverseProxy())

	n := negroni.Classic()
	n.UseHandler(r)

	log.Println("Listening on port "+ strconv.Itoa(portNumber))
	log.Fatal(http.ListenAndServe("0.0.0.0:"+strconv.Itoa(portNumber), n))

}
