package handlers

import (
	"net"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/gorilla/mux"
)

func NewMultipleHostReverseProxy() *httputil.ReverseProxy {
	director := func(req *http.Request) {
		v := mux.Vars(req)
		node := v["node"]
		if strings.HasPrefix(node, "ip") {
			// Node is actually an ip, need to convert underscores by dots.
			ip := strings.Replace(strings.TrimPrefix(node, "ip"), "_", ".", -1)

			if net.ParseIP(ip) == nil {
				// Not a valid IP, so treat this is a hostname.
			} else {
				node = ip
			}
		}

		// Validate that the node actually exists in the network
		// TODO:

		// Only proxy http for now
		req.URL.Scheme = "http"

		req.URL.Host = node
	}

	return &httputil.ReverseProxy{Director: director}
}
