package handlers

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"golang.org/x/crypto/acme/autocert"

	gh "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	lru "github.com/hashicorp/golang-lru"
	"github.com/play-with-docker/play-with-docker/config"
	"github.com/play-with-docker/play-with-docker/event"
	"github.com/play-with-docker/play-with-docker/pwd"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/negroni"
)

var core pwd.PWDApi
var e event.EventApi
var landings = map[string][]byte{}

type HandlerExtender func(h *mux.Router)

func Bootstrap(c pwd.PWDApi, ev event.EventApi) {
	core = c
	e = ev
}

func Register(extend HandlerExtender) {
	initLandings()

	r := mux.NewRouter()
	corsRouter := mux.NewRouter()

	corsHandler := gh.CORS(gh.AllowCredentials(), gh.AllowedHeaders([]string{"x-requested-with", "content-type"}), gh.AllowedMethods([]string{"GET", "POST", "HEAD", "DELETE"}), gh.AllowedOrigins([]string{"*"}))

	// Specific routes
	r.HandleFunc("/ping", Ping).Methods("GET")
	corsRouter.HandleFunc("/instances/images", GetInstanceImages).Methods("GET")
	corsRouter.HandleFunc("/sessions/{sessionId}", GetSession).Methods("GET")
	corsRouter.HandleFunc("/sessions/{sessionId}", CloseSession).Methods("DELETE")
	corsRouter.HandleFunc("/sessions/{sessionId}/setup", SessionSetup).Methods("POST")
	corsRouter.HandleFunc("/sessions/{sessionId}/instances", NewInstance).Methods("POST")
	corsRouter.HandleFunc("/sessions/{sessionId}/instances/{instanceName}/uploads", FileUpload).Methods("POST")
	corsRouter.HandleFunc("/sessions/{sessionId}/instances/{instanceName}", DeleteInstance).Methods("DELETE")
	corsRouter.HandleFunc("/sessions/{sessionId}/instances/{instanceName}/exec", Exec).Methods("POST")

	r.HandleFunc("/ooc", func(rw http.ResponseWriter, r *http.Request) {
		http.ServeFile(rw, r, "./www/ooc.html")
	}).Methods("GET")
	r.HandleFunc("/503", func(rw http.ResponseWriter, r *http.Request) {
		http.ServeFile(rw, r, "./www/503.html")
	}).Methods("GET")
	r.HandleFunc("/p/{sessionId}", Home).Methods("GET")
	r.PathPrefix("/assets").Handler(http.FileServer(http.Dir("./www")))
	r.HandleFunc("/robots.txt", func(rw http.ResponseWriter, r *http.Request) {
		http.ServeFile(rw, r, "www/robots.txt")
	})
	r.HandleFunc("/sdk.js", func(rw http.ResponseWriter, r *http.Request) {
		http.ServeFile(rw, r, "www/sdk.js")
	})

	corsRouter.HandleFunc("/sessions/{sessionId}/ws/", WSH)
	r.Handle("/metrics", promhttp.Handler())

	// Generic routes
	r.HandleFunc("/", Landing).Methods("GET")

	corsRouter.HandleFunc("/users/me", LoggedInUser).Methods("GET")
	r.HandleFunc("/users/{userId:^(?me)}", GetUser).Methods("GET")
	r.HandleFunc("/oauth/providers", ListProviders).Methods("GET")
	r.HandleFunc("/oauth/providers/{provider}/login", Login).Methods("GET")
	r.HandleFunc("/oauth/providers/{provider}/callback", LoginCallback).Methods("GET")
	r.HandleFunc("/playgrounds", NewPlayground).Methods("PUT")
	r.HandleFunc("/playgrounds", ListPlaygrounds).Methods("GET")
	r.HandleFunc("/my/playground", GetCurrentPlayground).Methods("GET")

	corsRouter.HandleFunc("/", NewSession).Methods("POST")

	if extend != nil {
		extend(corsRouter)
	}

	n := negroni.Classic()
	r.PathPrefix("/").Handler(negroni.New(negroni.Wrap(corsHandler(corsRouter))))
	n.UseHandler(r)

	httpServer := http.Server{
		Addr:              "0.0.0.0:" + config.PortNumber,
		Handler:           n,
		IdleTimeout:       30 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	if config.UseLetsEncrypt {
		domainCache, err := lru.New(5000)
		if err != nil {
			log.Fatalf("Could not start domain cache. Got: %v", err)
		}
		certManager := autocert.Manager{
			Prompt: autocert.AcceptTOS,
			HostPolicy: func(ctx context.Context, host string) error {
				if _, found := domainCache.Get(host); !found {
					if playground := core.PlaygroundFindByDomain(host); playground == nil {
						return fmt.Errorf("Playground for domain %s was not found", host)
					}
					domainCache.Add(host, true)
				}
				return nil
			},
			Cache: autocert.DirCache(config.LetsEncryptCertsDir),
		}

		httpServer.TLSConfig = &tls.Config{
			GetCertificate: certManager.GetCertificate,
		}

		go func() {
			rr := mux.NewRouter()
			rr.HandleFunc("/ping", Ping).Methods("GET")
			rr.Handle("/metrics", promhttp.Handler())
			rr.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
				http.Redirect(rw, r, fmt.Sprintf("https://%s", r.Host), http.StatusMovedPermanently)
			})
			nr := negroni.Classic()
			nr.UseHandler(rr)
			log.Println("Starting redirect server")
			redirectServer := http.Server{
				Addr:              "0.0.0.0:3001",
				Handler:           nr,
				IdleTimeout:       30 * time.Second,
				ReadHeaderTimeout: 5 * time.Second,
			}
			log.Fatal(redirectServer.ListenAndServe())
		}()

		log.Println("Listening on port " + config.PortNumber)
		log.Fatal(httpServer.ListenAndServeTLS("", ""))
	} else {
		log.Println("Listening on port " + config.PortNumber)
		log.Fatal(httpServer.ListenAndServe())
	}
}

func initLandings() {
	pgs, err := core.PlaygroundList()
	if err != nil {
		log.Fatal("Error getting playgrounds to initialize landings")
	}
	for _, p := range pgs {
		if p.AssetsDir == "" {
			p.AssetsDir = "default"
		}

		var b bytes.Buffer
		t, err := template.New("landing.html").Delims("[[", "]]").ParseFiles(fmt.Sprintf("./www/%s/landing.html", p.AssetsDir))
		if err != nil {
			log.Fatalf("Error parsing template %v", err)
		}
		if err := t.Execute(&b, struct{ SegmentId string }{config.SegmentId}); err != nil {
			log.Fatalf("Error executing template %v", err)
		}
		landingBytes, err := ioutil.ReadAll(&b)
		if err != nil {
			log.Fatalf("Error reading template bytes %v", err)
		}
		landings[p.Id] = landingBytes
	}
}
