package router

import (
	"fmt"
	"io"
	"log"
	"net"

	vhost "github.com/inconshreveable/go-vhost"
)

type Director func(host string) (*net.TCPAddr, error)

type proxyRouter struct {
	director Director
}

func (r *proxyRouter) Listen(laddr string) {
	l, err := net.Listen("tcp", laddr)
	defer l.Close()
	if err != nil {
		log.Fatal(err)
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go r.handleConnection(conn)
	}
}

func (r *proxyRouter) handleConnection(c net.Conn) {
	defer c.Close()
	// first try tls
	vhostConn, err := vhost.TLS(c)
	if err != nil {
		log.Printf("Incoming TLS connection produced an error. Error: %s", err)
		return
	}
	defer vhostConn.Close()

	host := vhostConn.ClientHelloMsg.ServerName
	c.LocalAddr()
	dstHost, err := r.director(fmt.Sprintf("%s:%d", host, 12))
	if err != nil {
		log.Printf("Error directing request: %v\n", err)
		return
	}

	d, err := net.Dial("tcp", dstHost.String())
	if err != nil {
		log.Printf("Error dialing backend %s: %v\n", dstHost.String(), err)
		return
	}

	errc := make(chan error, 2)
	cp := func(dst io.Writer, src io.Reader) {
		_, err := io.Copy(dst, src)
		errc <- err
	}
	go cp(d, vhostConn)
	go cp(vhostConn, d)
	<-errc
	/*
		req, err := http.ReadRequest(bufio.NewReader(c))
		if err != nil {
			log.Println(err)
			return
		}

		log.Println(req.Header)
	*/
}

func NewRouter(director Director) *proxyRouter {
	return &proxyRouter{director: director}
}

/*
	// Start the DNS server
	dns.HandleFunc(".", routerDns.DnsRequest)
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
	r := mux.NewRouter()
	tcpHandler := handlers.NewTCPProxy()
	r.Host(fmt.Sprintf("{subdomain:.*}pwd{node:%s}-{port:%s}.{tld:.*}", config.PWDHostnameRegex, config.PortRegex)).Handler(tcpHandler)
	r.Host(fmt.Sprintf("{subdomain:.*}pwd{node:%s}.{tld:.*}", config.PWDHostnameRegex)).Handler(tcpHandler)
	r.Host(fmt.Sprintf("pwd{alias:%s}-{session:%s}-{port:%s}.{tld:.*}", config.AliasnameRegex, config.AliasSessionRegex, config.PortRegex)).Handler(tcpHandler)
	r.Host(fmt.Sprintf("pwd{alias:%s}-{session:%s}.{tld:.*}", config.AliasnameRegex, config.AliasSessionRegex)).Handler(tcpHandler)
	r.HandleFunc("/ping", handlers.Ping).Methods("GET")
	n := negroni.Classic()
	n.UseHandler(r)

	httpServer := http.Server{
		Addr:              "0.0.0.0:" + config.PortNumber,
		Handler:           n,
		IdleTimeout:       30 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}
	// Now listen for TLS connections that need to be proxied
	tls.StartTLSProxy(config.SSLPortNumber)
	http.ListenAndServe()
*/
