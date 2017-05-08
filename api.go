package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"

	gh "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/miekg/dns"
	"github.com/play-with-docker/play-with-docker/config"
	"github.com/play-with-docker/play-with-docker/handlers"
	"github.com/play-with-docker/play-with-docker/services"
	"github.com/play-with-docker/play-with-docker/templates"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/negroni"
)

func main() {

	config.ParseFlags()

	bypassCaptcha := len(os.Getenv("GOOGLE_RECAPTCHA_DISABLED")) > 0

	// Start the DNS server
	dns.HandleFunc(".", handleDnsRequest)
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

	server := services.CreateWSServer()
	server.On("connection", handlers.WS)
	server.On("error", handlers.WSError)

	err := services.LoadSessionsFromDisk()
	if err != nil && !os.IsNotExist(err) {
		log.Fatal("Error decoding sessions from disk ", err)
	}

	r := mux.NewRouter()
	corsRouter := mux.NewRouter()

	// Reverse proxy (needs to be the first route, to make sure it is the first thing we check)
	//proxyHandler := handlers.NewMultipleHostReverseProxy()
	//websocketProxyHandler := handlers.NewMultipleHostWebsocketReverseProxy()

	tcpHandler := handlers.NewTCPProxy()

	corsHandler := gh.CORS(gh.AllowCredentials(), gh.AllowedHeaders([]string{"x-requested-with", "content-type"}), gh.AllowedOrigins([]string{"*"}))

	// Specific routes
	r.Host(`{subdomain:.*}{node:pwd[0-9]{1,3}_[0-9]{1,3}_[0-9]{1,3}_[0-9]{1,3}}-{port:[0-9]*}.{tld:.*}`).Handler(tcpHandler)
	r.Host(`{subdomain:.*}{node:pwd[0-9]{1,3}_[0-9]{1,3}_[0-9]{1,3}_[0-9]{1,3}}.{tld:.*}`).Handler(tcpHandler)
	r.HandleFunc("/ping", handlers.Ping).Methods("GET")
	corsRouter.HandleFunc("/instances/images", handlers.GetInstanceImages).Methods("GET")
	corsRouter.HandleFunc("/sessions/{sessionId}", handlers.GetSession).Methods("GET")
	corsRouter.HandleFunc("/sessions/{sessionId}/instances", handlers.NewInstance).Methods("POST")
	corsRouter.HandleFunc("/sessions/{sessionId}/instances/{instanceName}", handlers.DeleteInstance).Methods("DELETE")
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

	go func() {
		log.Println("Listening on port " + config.PortNumber)
		log.Fatal(http.ListenAndServe("0.0.0.0:"+config.PortNumber, n))
	}()

	ssl := mux.NewRouter()
	sslProxyHandler := handlers.NewSSLDaemonHandler()
	ssl.Host(`{subdomain:.*}{node:pwd[0-9]{1,3}_[0-9]{1,3}_[0-9]{1,3}_[0-9]{1,3}}-2375.{tld:.*}`).Handler(sslProxyHandler)
	log.Println("Listening TLS on port " + config.SSLPortNumber)

	s := &http.Server{Addr: "0.0.0.0:" + config.SSLPortNumber, Handler: ssl}
	s.TLSConfig = &tls.Config{}
	s.TLSConfig.GetCertificate = func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {

		chunks := strings.Split(clientHello.ServerName, ".")
		chunks = strings.Split(chunks[0], "-")
		ip := strings.Replace(strings.TrimPrefix(chunks[0], "pwd"), "_", ".", -1)
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

var dnsFilter = regexp.MustCompile(`pwd[0-9]{1,3}_[0-9]{1,3}_[0-9]{1,3}_[0-9]{1,3}`)

func handleDnsRequest(w dns.ResponseWriter, r *dns.Msg) {
	if len(r.Question) > 0 && dnsFilter.MatchString(r.Question[0].Name) {
		// this is something we know about and we should try to handle
		question := r.Question[0].Name
		domainChunks := strings.Split(question, ".")
		tldChunks := strings.Split(strings.TrimPrefix(domainChunks[0], "pwd"), "-")
		ip := strings.Replace(tldChunks[0], "_", ".", -1)

		m := new(dns.Msg)
		m.SetReply(r)
		m.Authoritative = true
		m.RecursionAvailable = true
		a, err := dns.NewRR(fmt.Sprintf("%s 60 IN A %s", question, ip))
		if err != nil {
			log.Fatal(err)
		}
		m.Answer = append(m.Answer, a)
		w.WriteMsg(m)
		return
	} else {
		if len(r.Question) > 0 {
			question := r.Question[0].Name

			if question == "localhost." {
				log.Printf("Not a PWD host. Asked for [localhost.] returning automatically [127.0.0.1]\n")
				m := new(dns.Msg)
				m.SetReply(r)
				m.Authoritative = true
				m.RecursionAvailable = true
				a, err := dns.NewRR(fmt.Sprintf("%s 60 IN A 127.0.0.1", question))
				if err != nil {
					log.Fatal(err)
				}
				m.Answer = append(m.Answer, a)
				w.WriteMsg(m)
				return
			}

			log.Printf("Not a PWD host. Looking up [%s]\n", question)
			ips, err := net.LookupIP(question)
			if err != nil {
				// we have no information about this and we are not a recursive dns server, so we just fail so the client can fallback to the next dns server it has configured
				w.Close()
				// dns.HandleFailed(w, r)
				return
			}
			log.Printf("Not a PWD host. Looking up [%s] got [%s]\n", question, ips)
			m := new(dns.Msg)
			m.SetReply(r)
			m.Authoritative = true
			m.RecursionAvailable = true
			for _, ip := range ips {
				ipv4 := ip.To4()
				if ipv4 == nil {
					a, err := dns.NewRR(fmt.Sprintf("%s 60 IN AAAA %s", question, ip.String()))
					if err != nil {
						log.Fatal(err)
					}
					m.Answer = append(m.Answer, a)
				} else {
					a, err := dns.NewRR(fmt.Sprintf("%s 60 IN A %s", question, ipv4.String()))
					if err != nil {
						log.Fatal(err)
					}
					m.Answer = append(m.Answer, a)
				}
			}
			w.WriteMsg(m)
			return

		} else {
			log.Printf("Not a PWD host. Got DNS without any question\n")
			// we have no information about this and we are not a recursive dns server, so we just fail so the client can fallback to the next dns server it has configured
			w.Close()
			// dns.HandleFailed(w, r)
			return
		}
	}
}
