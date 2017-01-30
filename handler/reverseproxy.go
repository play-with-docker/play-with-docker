package handler

import "net/http/httputil"

func (h *handlers) newMultipleHostReverseProxy() *httputil.ReverseProxy {
	director := h.core.NewHTTPDirector()

	return &httputil.ReverseProxy{Director: director}
}

func (h *handlers) newSSLDaemonHandler() *httputil.ReverseProxy {
	director := h.core.NewDockerDaemonDirector()

	return &httputil.ReverseProxy{Director: director}
}
