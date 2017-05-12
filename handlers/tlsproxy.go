package handlers

import (
	"fmt"
	"io"
	"log"
	"net"
	"regexp"
	"strings"

	vhost "github.com/inconshreveable/go-vhost"
	"github.com/play-with-docker/play-with-docker/config"
)

func StartTLSProxy(port string) {
	var validProxyHost = regexp.MustCompile(`^.*pwd([0-9]{1,3}-[0-9]{1,3}-[0-9]{1,3}-[0-9]{1,3}(?:-[0-9]{1,5})?)\..*$`)

	tlsListener, tlsErr := net.Listen("tcp", fmt.Sprintf(":%s", port))
	log.Println("Listening on port " + port)
	if tlsErr != nil {
		log.Fatal(tlsErr)
	}
	defer tlsListener.Close()
	for {
		// Wait for TLS Connection
		conn, err := tlsListener.Accept()
		if err != nil {
			log.Printf("Could not accept new TLS connection. Error: %s", err)
			continue
		}
		// Handle connection on a new goroutine and continue accepting other new connections
		go func(c net.Conn) {
			defer c.Close()
			vhostConn, err := vhost.TLS(conn)
			if err != nil {
				log.Printf("Incoming TLS connection produced an error. Error: %s", err)
				return
			}
			defer vhostConn.Close()

			host := vhostConn.ClientHelloMsg.ServerName
			match := validProxyHost.FindStringSubmatch(host)
			if len(match) == 2 {
				// This is a valid proxy host, keep only the important part
				host = match[1]
			} else {
				// Not a valid proxy host, just close connection.
				return
			}

			var targetIP string
			targetPort := "443"

			hostPort := strings.Split(host, ":")
			if len(hostPort) > 1 && hostPort[1] != config.SSLPortNumber {
				targetPort = hostPort[1]
			}

			target := strings.Split(hostPort[0], "-")
			if len(target) > 1 {
				targetPort = target[1]
			}

			ip := strings.Replace(target[0], "_", ".", -1)

			if net.ParseIP(ip) == nil {
				// Not a valid IP, so treat this is a hostname.
			} else {
				targetIP = ip
			}

			dest := fmt.Sprintf("%s:%s", targetIP, targetPort)
			d, err := net.Dial("tcp", dest)
			if err != nil {
				log.Printf("Error dialing backend %s: %v\n", dest, err)
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
		}(conn)
	}

}
