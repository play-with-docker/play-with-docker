package handlers

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/franela/play-with-docker/config"
	"github.com/gorilla/mux"
)

func getTargetInfo(vars map[string]string, req *http.Request) (string, string) {
	node := vars["node"]
	port := vars["port"]
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

	return node, port

}

func NewMultipleHostReverseProxy() *httputil.ReverseProxy {
	var transport http.RoundTripper = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 0,
		}).DialContext,
		DisableKeepAlives:     true,
		MaxIdleConns:          1,
		IdleConnTimeout:       100 * time.Millisecond,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	}
	director := func(req *http.Request) {
		v := mux.Vars(req)
		node, port := getTargetInfo(v, req)

		if port == "443" {
			// Only proxy http for now
			req.URL.Scheme = "https"
		} else {
			// Only proxy http for now
			req.URL.Scheme = "http"
		}

		req.URL.Host = fmt.Sprintf("%s:%s", node, port)
	}

	return &httputil.ReverseProxy{Director: director, Transport: transport}
}

func NewSSLDaemonHandler() *httputil.ReverseProxy {
	director := func(req *http.Request) {
		v := mux.Vars(req)
		node := v["node"]
		if strings.HasPrefix(node, "pwd") {
			// Node is actually an ip, need to convert underscores by dots.
			ip := strings.Replace(strings.TrimPrefix(node, "pwd"), "_", ".", -1)

			if net.ParseIP(ip) == nil {
				// Not a valid IP, so treat this is a hostname.
			} else {
				node = ip
			}
		}

		// Only proxy http for now
		req.URL.Scheme = "http"

		req.URL.Host = fmt.Sprintf("%s:%s", node, "2375")
	}

	return &httputil.ReverseProxy{Director: director}
}
