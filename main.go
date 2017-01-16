package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"flag"
	"strconv"

	"github.com/franela/play-with-docker/core"
	"github.com/franela/play-with-docker/handler"
	"github.com/franela/play-with-docker/services"
	"github.com/urfave/negroni"
)

func main() {
	var sslPortNumber, portNumber int
	flag.IntVar(&portNumber, "port", 3000, "Give a TCP port to run the application")
	flag.IntVar(&sslPortNumber, "sslPort", 3001, "Give a SSL TCP port")
	flag.Parse()

	bypassCaptcha := len(os.Getenv("GOOGLE_RECAPTCHA_DISABLED")) > 0

	err := services.LoadSessionsFromDisk()
	if err != nil && !os.IsNotExist(err) {
		log.Fatal("Error decoding sessions from disk ", err)
	}

	conf := handler.NewConfig()
	conf.BypassCaptcha = bypassCaptcha
	r, err := handler.New(core.New(), conf)
	if err != nil {
		log.Fatal(err)
	}
	n := negroni.Classic()
	n.UseHandler(r)

	go func() {
		log.Println("Listening on port " + strconv.Itoa(portNumber))
		log.Fatal(http.ListenAndServe("0.0.0.0:"+strconv.Itoa(portNumber), n))
	}()

	rssl, err := handler.NewSSL()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Listening TLS on port " + strconv.Itoa(sslPortNumber))

	s := &http.Server{Addr: "0.0.0.0:" + strconv.Itoa(sslPortNumber), Handler: rssl}
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
