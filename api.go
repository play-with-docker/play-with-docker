package main

import (
	"log"
	"net/http"
	"os"

	"flag"
	"strconv"

	"github.com/franela/play-with-docker/handlers"
	"github.com/franela/play-with-docker/services"
	"github.com/franela/play-with-docker/templates"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/negroni"
)

func main() {
	var sslPortNumber, portNumber int
	var key, cert string
	flag.IntVar(&portNumber, "port", 3000, "Give a TCP port to run the application")
	flag.IntVar(&sslPortNumber, "sslPort", 3001, "Give a SSL TCP port")
	flag.StringVar(&key, "key", "./pwd/server-key.pem", "Server key for SSL")
	flag.StringVar(&cert, "cert", "./pwd/server.pem", "Give a SSL cert")
	flag.Parse()

	bypassCaptcha := len(os.Getenv("GOOGLE_RECAPTCHA_DISABLED")) > 0

	server := services.CreateWSServer()
	server.On("connection", handlers.WS)
	server.On("error", handlers.WSError)

	err := services.LoadSessionsFromDisk()
	if err != nil && !os.IsNotExist(err) {
		log.Fatal("Error decoding sessions from disk ", err)
	}

	r := mux.NewRouter()

	// Reverse proxy (needs to be the first route, to make sure it is the first thing we check)
	proxyHandler := handlers.NewMultipleHostReverseProxy()

	// Specific routes
	r.Host(`{node:ip[0-9]{1,3}_[0-9]{1,3}_[0-9]{1,3}_[0-9]{1,3}}-{port:[0-9]*}.{tld:.*}`).Handler(proxyHandler)
	r.Host(`{node:ip[0-9]{1,3}_[0-9]{1,3}_[0-9]{1,3}_[0-9]{1,3}}.{tld:.*}`).Handler(proxyHandler)
	r.HandleFunc("/ping", handlers.Ping).Methods("GET")
	r.HandleFunc("/sessions/{sessionId}", handlers.GetSession).Methods("GET")
	r.HandleFunc("/sessions/{sessionId}/instances", handlers.NewInstance).Methods("POST")
	r.HandleFunc("/sessions/{sessionId}/instances/{instanceName}", handlers.DeleteInstance).Methods("DELETE")
	r.HandleFunc("/keys", handlers.GetKeys)

	h := func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./www/index.html")
	}

	r.HandleFunc("/p/{sessionId}", h).Methods("GET")
	r.PathPrefix("/assets").Handler(http.FileServer(http.Dir("./www")))
	r.HandleFunc("/robots.txt", func(rw http.ResponseWriter, r *http.Request) {
		http.ServeFile(rw, r, "www/robots.txt")
	})

	r.Handle("/sessions/{sessionId}/ws/", server)
	r.Handle("/metrics", promhttp.Handler())

	// Generic routes
	r.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		if bypassCaptcha {
			http.ServeFile(rw, r, "./www/bypass.html")
		} else {
			welcome, tmplErr := templates.GetWelcomeTemplate()
			if tmplErr != nil {
				log.Fatal(tmplErr)
			}
			rw.Write(welcome)
		}
	}).Methods("GET")

	r.HandleFunc("/", handlers.NewSession).Methods("POST")

	n := negroni.Classic()
	n.UseHandler(r)

	go func() {
		log.Println("Listening on port " + strconv.Itoa(portNumber))
		log.Fatal(http.ListenAndServe("0.0.0.0:"+strconv.Itoa(portNumber), n))
	}()

	log.Println("Listening TLS on port " + strconv.Itoa(sslPortNumber))
	log.Fatal(http.ListenAndServeTLS("0.0.0.0:"+strconv.Itoa(sslPortNumber), cert, key, n))

}
