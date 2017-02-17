package handlers

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/franela/play-with-docker/config"
	"github.com/gorilla/mux"
	"github.com/yhat/wsutil"
)

func NewMultipleHostWebsocketReverseProxy() *wsutil.ReverseProxy {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	director := func(req *http.Request) {
		v := mux.Vars(req)
		node := v["node"]
		port := v["port"]
		hostPort := strings.Split(req.Host, ":")

		// give priority to the URL host port
		if len(hostPort) > 1 && hostPort[1] != config.PortNumber {
			port = hostPort[1]
		} else if port == "" {
			port = "80"
		}

		if strings.HasPrefix(node, "pwd") {
			// Node is actually an ip, need to convert underscores by dots.
			ip := strings.Replace(strings.TrimPrefix(node, "pwd"), "_", ".", -1)

			if net.ParseIP(ip) == nil {
				// Not a valid IP, so treat this is a hostname.
			} else {
				node = ip
			}
		}

		if port == "443" {
			// Only proxy http for now
			req.URL.Scheme = "wss"
		} else {
			// Only proxy http for now
			req.URL.Scheme = "ws"
		}
		req.URL.Host = fmt.Sprintf("%s:%s", node, port)
	}

	return &wsutil.ReverseProxy{Director: director, TLSClientConfig: tlsConfig}
}
