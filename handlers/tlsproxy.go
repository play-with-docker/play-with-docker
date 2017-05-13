package handlers

import (
	"fmt"
	"io"
	"log"
	"net"
	"regexp"
	"strings"

	vhost "github.com/inconshreveable/go-vhost"
	"github.com/play-with-docker/play-with-docker/services"
)

func StartTLSProxy(port string) {
	var validProxyHost = regexp.MustCompile(`^.*pwd([0-9]{1,3}-[0-9]{1,3}-[0-9]{1,3}-[0-9]{1,3})(?:-?([0-9]{1,5}))?\..*$`)
	var validAliasProxyHost = regexp.MustCompile(`^.*pwd([0-9|a-z|A-Z]*)-([0-9|a-z|A-Z]{8})(?:-?([0-9]{1,5}))?\..*$`)

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

			var targetIP string
			targetPort := "443"

			host := vhostConn.ClientHelloMsg.ServerName
			match := validProxyHost.FindStringSubmatch(host)
			if len(match) < 2 {
				// Not a valid proxy host, try alias hosts
				match := validAliasProxyHost.FindStringSubmatch(host)
				if len(match) < 4 {
					// Not valid, just close the connection
					return
				} else {
					alias := match[1]
					sessionPrefix := match[2]
					instance := services.FindInstanceByAlias(sessionPrefix, alias)
					if instance != nil {
						targetIP = instance.IP
					} else {
						return
					}
					if len(match) == 4 {
						targetPort = match[3]
					}
				}
			} else {
				// Valid proxy host
				ip := strings.Replace(match[1], "-", ".", -1)
				if net.ParseIP(ip) == nil {
					// Not a valid IP, so treat this is a hostname.
					return
				} else {
					targetIP = ip
				}
				if len(match) == 3 {
					targetPort = match[2]
				}
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
