package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/franela/play-with-docker/config"
	"github.com/franela/play-with-docker/handlers"
	"github.com/franela/play-with-docker/services"
	"github.com/franela/play-with-docker/templates"
	gh "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/negroni"
)

func main() {

	config.ParseFlags()

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
	r.Handle("/sessions/{sessionId}/instances", http.HandlerFunc(handlers.NewInstance)).Methods("POST")
	r.HandleFunc("/sessions/{sessionId}/instances/{instanceName}", handlers.DeleteInstance).Methods("DELETE")
	r.HandleFunc("/sessions/{sessionId}/instances/{instanceName}/keys", handlers.SetKeys).Methods("POST")

	h := func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./www/index.html")
	}

	r.HandleFunc("/p/{sessionId}", h).Methods("GET")
	r.PathPrefix("/assets").Handler(http.FileServer(http.Dir("./www")))
	r.HandleFunc("/robots.txt", func(rw http.ResponseWriter, r *http.Request) {
		http.ServeFile(rw, r, "www/robots.txt")
	})
	r.HandleFunc("/sdk.js", func(rw http.ResponseWriter, r *http.Request) {
		http.ServeFile(rw, r, "www/sdk.js")
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
		log.Println("Listening on port " + config.PortNumber)
		log.Fatal(http.ListenAndServe("0.0.0.0:"+config.PortNumber, gh.CORS(gh.AllowCredentials(), gh.AllowedHeaders([]string{"x-requested-with", "content-type"}), gh.AllowedOrigins([]string{"*"}))(n)))
	}()

	ssl := mux.NewRouter()
	sslProxyHandler := handlers.NewSSLDaemonHandler()
	ssl.Host(`{node:ip[0-9]{1,3}_[0-9]{1,3}_[0-9]{1,3}_[0-9]{1,3}}-2375.{tld:.*}`).Handler(sslProxyHandler)
	log.Println("Listening TLS on port " + config.SSLPortNumber)

	s := &http.Server{Addr: "0.0.0.0:" + config.SSLPortNumber, Handler: ssl}
	s.TLSConfig = &tls.Config{}
	s.TLSConfig.GetCertificate = func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {

		chunks := strings.Split(clientHello.ServerName, ".")
		chunks = strings.Split(chunks[0], "-")
		ip := strings.Replace(strings.TrimPrefix(chunks[0], "ip"), "_", ".", -1)
		i := services.FindInstanceByIP(ip)
		if i == nil {
			return nil, fmt.Errorf("Instance %s doesn't exist", clientHello.ServerName)
		}
		if i.GetCertificate() == nil {
			return nil, fmt.Errorf("Instance %s doesn't have a certificate", clientHello.ServerName)
		}
		return i.GetCertificate(), nil
	}
	log.Fatal(s.ListenAndServeTLS("", ""))
}
