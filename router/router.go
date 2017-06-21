package router

import (
	"bufio"
	"io"
	"log"
	"net"
	"net/http"
	"sync"

	vhost "github.com/inconshreveable/go-vhost"
)

type Director func(host string) (*net.TCPAddr, error)

type proxyRouter struct {
	sync.Mutex

	director Director
	listener net.Listener
	closed   bool
}

func (r *proxyRouter) Listen(laddr string) {
	l, err := net.Listen("tcp", laddr)
	if err != nil {
		log.Fatal(err)
	}
	r.listener = l
	go func() {
		for !r.closed {
			conn, err := r.listener.Accept()
			if err != nil {
				continue
			}
			go r.handleConnection(conn)
		}
	}()
}

func (r *proxyRouter) Close() {
	r.Lock()
	defer r.Unlock()

	if r.listener != nil {
		r.listener.Close()
	}
	r.closed = true
}

func (r *proxyRouter) ListenAddress() string {
	if r.listener != nil {
		return r.listener.Addr().String()
	}
	return ""
}

func (r *proxyRouter) handleConnection(c net.Conn) {
	defer c.Close()
	// first try tls
	vhostConn, err := vhost.TLS(c)

	if err == nil {
		// It is a TLS connection
		defer vhostConn.Close()
		host := vhostConn.ClientHelloMsg.ServerName
		dstHost, err := r.director(host)
		if err != nil {
			log.Printf("Error directing request: %v\n", err)
			return
		}
		d, err := net.Dial("tcp", dstHost.String())
		if err != nil {
			log.Printf("Error dialing backend %s: %v\n", dstHost.String(), err)
			return
		}

		proxy(vhostConn, d)
	} else {
		// it is not TLS
		// treat it as an http connection

		req, err := http.ReadRequest(bufio.NewReader(vhostConn))
		if err != nil {
			// It is not http neither. So just close the connection.
			return
		}
		dstHost, err := r.director(req.Host)
		if err != nil {
			log.Printf("Error directing request: %v\n", err)
			return
		}
		d, err := net.Dial("tcp", dstHost.String())
		if err != nil {
			log.Printf("Error dialing backend %s: %v\n", dstHost.String(), err)
			return
		}
		err = req.Write(d)
		if err != nil {
			log.Printf("Error requesting backend %s: %v\n", dstHost.String(), err)
			return
		}
		proxy(c, d)
	}
}

func proxy(src, dst net.Conn) {
	errc := make(chan error, 2)
	cp := func(dst io.Writer, src io.Reader) {
		_, err := io.Copy(dst, src)
		errc <- err
	}
	go cp(src, dst)
	go cp(dst, src)
	<-errc
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
