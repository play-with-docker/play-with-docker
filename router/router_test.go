package router

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func getRouterUrl(scheme string, r *proxyRouter) string {
	chunks := strings.Split(r.ListenAddress(), ":")
	return fmt.Sprintf("%s://localhost:%s", scheme, chunks[len(chunks)-1])
}

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
	r.Listen(":0")
	defer r.Close()

	req, err := http.NewRequest("GET", getRouterUrl("https", r), nil)
	assert.Nil(t, err)

	resp, err := client.Do(req)
	assert.Nil(t, err)

	body, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, msg, string(body))
	assert.Equal(t, "localhost", receivedHost)
}

func TestProxy_Http(t *testing.T) {
	const msg = "It works!"

	var receivedHost string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, msg)
	}))
	defer ts.Close()

	r := NewRouter(func(host string) (*net.TCPAddr, error) {
		receivedHost = host
		u, _ := url.Parse(ts.URL)
		a, _ := net.ResolveTCPAddr("tcp", u.Host)
		return a, nil
	})
	r.Listen(":0")
	defer r.Close()

	req, err := http.NewRequest("GET", getRouterUrl("http", r), nil)
	assert.Nil(t, err)

	resp, err := http.DefaultClient.Do(req)
	assert.Nil(t, err)

	body, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, msg, string(body))

	u, _ := url.Parse(getRouterUrl("http", r))
	assert.Equal(t, u.Host, receivedHost)
}

func TestProxy_WS(t *testing.T) {
	const msg = "It works!"

	var serverReceivedMessage string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var upgrader = websocket.Upgrader{}
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Print("upgrade:", err)
			return
		}
		defer c.Close()
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return
		}
		serverReceivedMessage = string(message)
		err = c.WriteMessage(mt, message)
		if err != nil {
			log.Println("write:", err)
			return
		}
	}))
	defer ts.Close()

	r := NewRouter(func(host string) (*net.TCPAddr, error) {
		u, _ := url.Parse(ts.URL)
		a, _ := net.ResolveTCPAddr("tcp", u.Host)
		return a, nil
	})
	r.Listen(":0")
	defer r.Close()

	c, _, err := websocket.DefaultDialer.Dial(getRouterUrl("ws", r), nil)
	assert.Nil(t, err)
	defer c.Close()

	var clientReceivedMessage []byte
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		_, clientReceivedMessage, _ = c.ReadMessage()
		wg.Done()
	}()
	err = c.WriteMessage(websocket.TextMessage, []byte(msg))
	assert.Nil(t, err)

	wg.Wait()

	assert.Equal(t, msg, string(clientReceivedMessage))
	assert.Equal(t, msg, serverReceivedMessage)
}

func TestProxy_WSS(t *testing.T) {
	const msg = "It works!"

	var serverReceivedMessage string

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var upgrader = websocket.Upgrader{}
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Print("upgrade:", err)
			return
		}
		defer c.Close()
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return
		}
		serverReceivedMessage = string(message)
		err = c.WriteMessage(mt, message)
		if err != nil {
			log.Println("write:", err)
			return
		}
	}))
	defer ts.Close()

	r := NewRouter(func(host string) (*net.TCPAddr, error) {
		u, _ := url.Parse(ts.URL)
		a, _ := net.ResolveTCPAddr("tcp", u.Host)
		return a, nil
	})
	r.Listen(":0")
	defer r.Close()

	d := websocket.Dialer{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	c, _, err := d.Dial(getRouterUrl("wss", r), nil)
	assert.Nil(t, err)
	defer c.Close()

	var clientReceivedMessage []byte

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		_, clientReceivedMessage, _ = c.ReadMessage()
		wg.Done()
	}()

	err = c.WriteMessage(websocket.TextMessage, []byte(msg))
	assert.Nil(t, err)

	wg.Wait()

	assert.Equal(t, msg, string(clientReceivedMessage))
	assert.Equal(t, msg, serverReceivedMessage)
}
