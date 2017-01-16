package handler

import (
	"fmt"
	"log"
	"net/http"

	"github.com/franela/play-with-docker/core"
	"github.com/franela/play-with-docker/recaptcha"
	"github.com/franela/play-with-docker/templates"
	"github.com/googollee/go-socket.io"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type handlers struct {
	core      core.Core
	recaptcha recaptcha.Recaptcha
}

type config struct {
	BypassCaptcha bool
	RootPath      string
}

func NewConfig() config {
	return config{RootPath: ".", BypassCaptcha: false}
}

func NewSSL() (http.Handler, error) {
	ssl := mux.NewRouter()
	sslProxyHandler := NewSSLDaemonHandler()
	ssl.Host(`{node:ip[0-9]{1,3}_[0-9]{1,3}_[0-9]{1,3}_[0-9]{1,3}}-2375.{tld:.*}`).Handler(sslProxyHandler)

	return ssl, nil
}

func New(conf config, c core.Core, recap recaptcha.Recaptcha) (http.Handler, error) {
	h := &handlers{core: c, recaptcha: recap}
	r := mux.NewRouter()

	// WebSocket support
	wsServer, err := socketio.NewServer(nil)
	if err != nil {
		return nil, err
	}
	wsServer.On("connection", h.ws)
	wsServer.On("error", h.wsError)

	// Reverse proxy (needs to be the first route, to make sure it is the first thing we check)
	proxyHandler := NewMultipleHostReverseProxy()

	// Specific routes
	r.Host(`{node:ip[0-9]{1,3}_[0-9]{1,3}_[0-9]{1,3}_[0-9]{1,3}}-{port:[0-9]*}.{tld:.*}`).Handler(proxyHandler)
	r.Host(`{node:ip[0-9]{1,3}_[0-9]{1,3}_[0-9]{1,3}_[0-9]{1,3}}.{tld:.*}`).Handler(proxyHandler)

	r.HandleFunc("/ping", h.ping).Methods("GET")
	r.HandleFunc("/sessions/{sessionId}", h.getSession).Methods("GET")
	r.HandleFunc("/sessions/{sessionId}/instances", h.newInstance).Methods("POST")
	r.HandleFunc("/sessions/{sessionId}/instances/{instanceName}", h.deleteInstance).Methods("DELETE")
	r.HandleFunc("/sessions/{sessionId}/instances/{instanceName}/keys", h.setKeys).Methods("POST")

	serveIndex := func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, fmt.Sprintf("%s/www/index.html", conf.RootPath))
	}

	r.HandleFunc("/p/{sessionId}", serveIndex).Methods("GET")
	r.PathPrefix("/assets").Handler(http.FileServer(http.Dir("./www")))
	r.HandleFunc("/robots.txt", func(rw http.ResponseWriter, r *http.Request) {
		http.ServeFile(rw, r, "www/robots.txt")
	})

	r.Handle("/sessions/{sessionId}/ws/", wsServer)
	r.Handle("/metrics", promhttp.Handler())

	// Generic routes
	r.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		if conf.BypassCaptcha {
			http.ServeFile(rw, r, fmt.Sprintf("%s/www/bypass.html", conf.RootPath))
		} else {
			welcome, tmplErr := templates.GetWelcomeTemplate(conf.RootPath)
			if tmplErr != nil {
				log.Fatal(tmplErr)
			}
			rw.Write(welcome)
		}
	}).Methods("GET")

	r.HandleFunc("/", h.newSession).Methods("POST")

	return r, nil
}
