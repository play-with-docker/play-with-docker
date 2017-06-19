package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	gh "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/miekg/dns"
	"github.com/play-with-docker/play-with-docker/config"
	"github.com/play-with-docker/play-with-docker/handlers"
	"github.com/play-with-docker/play-with-docker/templates"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/negroni"
)

func main() {

	config.ParseFlags()
	handlers.Bootstrap()

	bypassCaptcha := len(os.Getenv("GOOGLE_RECAPTCHA_DISABLED")) > 0

	// Start the DNS server
	dns.HandleFunc(".", handlers.DnsRequest)
	udpDnsServer := &dns.Server{Addr: ":53", Net: "udp"}
	go func() {
		err := udpDnsServer.ListenAndServe()
		if err != nil {
			log.Fatal(err)
		}
	}()
	tcpDnsServer := &dns.Server{Addr: ":53", Net: "tcp"}
	go func() {
		err := tcpDnsServer.ListenAndServe()
		if err != nil {
			log.Fatal(err)
		}
	}()

	server := handlers.Broadcast.GetHandler()

	r := mux.NewRouter()
	corsRouter := mux.NewRouter()

	// Reverse proxy (needs to be the first route, to make sure it is the first thing we check)
	//proxyHandler := handlers.NewMultipleHostReverseProxy()
	//websocketProxyHandler := handlers.NewMultipleHostWebsocketReverseProxy()

	tcpHandler := handlers.NewTCPProxy()

	corsHandler := gh.CORS(gh.AllowCredentials(), gh.AllowedHeaders([]string{"x-requested-with", "content-type"}), gh.AllowedOrigins([]string{"*"}))

	// Specific routes
	r.Host(fmt.Sprintf("{subdomain:.*}pwd{node:%s}-{port:%s}.{tld:.*}", config.PWDHostnameRegex, config.PortRegex)).Handler(tcpHandler)
	r.Host(fmt.Sprintf("{subdomain:.*}pwd{node:%s}.{tld:.*}", config.PWDHostnameRegex)).Handler(tcpHandler)
	r.Host(fmt.Sprintf("pwd{alias:%s}-{session:%s}-{port:%s}.{tld:.*}", config.AliasnameRegex, config.AliasSessionRegex, config.PortRegex)).Handler(tcpHandler)
	r.Host(fmt.Sprintf("pwd{alias:%s}-{session:%s}.{tld:.*}", config.AliasnameRegex, config.AliasSessionRegex)).Handler(tcpHandler)
	r.HandleFunc("/ping", handlers.Ping).Methods("GET")
	corsRouter.HandleFunc("/instances/images", handlers.GetInstanceImages).Methods("GET")
	corsRouter.HandleFunc("/sessions/{sessionId}", handlers.GetSession).Methods("GET")
	corsRouter.HandleFunc("/sessions/{sessionId}/setup", handlers.SessionSetup).Methods("POST")
	corsRouter.HandleFunc("/sessions/{sessionId}/instances", handlers.NewInstance).Methods("POST")
	corsRouter.HandleFunc("/sessions/{sessionId}/instances/{instanceName}/uploads", handlers.FileUpload).Methods("POST")
	corsRouter.HandleFunc("/sessions/{sessionId}/instances/{instanceName}", handlers.DeleteInstance).Methods("DELETE")
	corsRouter.HandleFunc("/sessions/{sessionId}/instances/{instanceName}/exec", handlers.Exec).Methods("POST")

	r.HandleFunc("/p/{sessionId}", handlers.Home).Methods("GET")
	r.PathPrefix("/assets").Handler(http.FileServer(http.Dir("./www")))
	r.HandleFunc("/robots.txt", func(rw http.ResponseWriter, r *http.Request) {
		http.ServeFile(rw, r, "www/robots.txt")
	})
	r.HandleFunc("/sdk.js", func(rw http.ResponseWriter, r *http.Request) {
		http.ServeFile(rw, r, "www/sdk.js")
	})

	corsRouter.Handle("/sessions/{sessionId}/ws/", server)
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

	corsRouter.HandleFunc("/", handlers.NewSession).Methods("POST")

	n := negroni.Classic()
	r.PathPrefix("/").Handler(negroni.New(negroni.Wrap(corsHandler(corsRouter))))
	n.UseHandler(r)

	httpServer := http.Server{
		Addr:              "0.0.0.0:" + config.PortNumber,
		Handler:           n,
		IdleTimeout:       30 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Println("Listening on port " + config.PortNumber)
		log.Fatal(httpServer.ListenAndServe())
	}()

	go handlers.ListenSSHProxy("0.0.0.0:1022")

	// Now listen for TLS connections that need to be proxied
	handlers.StartTLSProxy(config.SSLPortNumber)
}
