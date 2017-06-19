package router

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProxy_TLS(t *testing.T) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	const msg = "It works!"

	var receivedHost string

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, msg)
	}))
	defer ts.Close()

	r := NewRouter(func(host string) (*net.TCPAddr, error) {
		receivedHost = host
		u, _ := url.Parse(ts.URL)
		a, _ := net.ResolveTCPAddr("tcp", u.Host)
		return a, nil
	})
	go r.Listen(":8080")

	req, err := http.NewRequest("GET", "https://localhost:8080", nil)
	assert.Nil(t, err)

	resp, err := client.Do(req)
	assert.Nil(t, err)

	body, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, msg, string(body))
	assert.Equal(t, "localhost:8080", receivedHost)
}
