package handlers

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/yhat/wsutil"
)

func NewMultipleHostWebsocketReverseProxy() *wsutil.ReverseProxy {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	director := func(req *http.Request) {
		v := mux.Vars(req)

		node, port, host := getTargetInfo(v, req)

		if port == "443" {
			// Only proxy http for now
			req.URL.Scheme = "wss"
		} else {
			// Only proxy http for now
			req.URL.Scheme = "ws"
		}
		req.Host = host
		req.URL.Host = fmt.Sprintf("%s:%s", node, port)
	}

	return &wsutil.ReverseProxy{Director: director, TLSClientConfig: tlsConfig}
}
