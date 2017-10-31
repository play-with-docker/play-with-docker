package router

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/gorilla/websocket"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func testSshClient(user string, r *proxyRouter) error {
	reader := rand.Reader
	bitSize := 2048

	key, err := rsa.GenerateKey(reader, bitSize)
	if err != nil {
		return err
	}
	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		return err
	}
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}
	chunks := strings.Split(r.ListenSshAddress(), ":")
	l := fmt.Sprintf("127.0.0.1:%s", chunks[len(chunks)-1])
	client, err := ssh.Dial("tcp", l, config)
	if err != nil {
		return err
	}
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	return nil
}

func testSshServer(f func(user, pass, ctype string)) (string, error) {
	reader := rand.Reader
	bitSize := 2048

	key, err := rsa.GenerateKey(reader, bitSize)
	if err != nil {
		return "", err
	}
	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		return "", err
	}

	var receivedUser string
	var receivedPass string
	var receivedChannelType string

	config := &ssh.ServerConfig{
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			receivedUser = conn.User()
			receivedPass = string(password)

			return nil, nil
		},
	}
	config.AddHostKey(signer)
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		return "", err
	}

	go func() {
		defer listener.Close()
		nConn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			return
		}
		conn, chans, reqs, err := ssh.NewServerConn(nConn, config)
		if err != nil {
			log.Println(err)
			return
		}
		go ssh.DiscardRequests(reqs)
		defer nConn.Close()
		defer conn.Close()

		ch := <-chans

		receivedChannelType = ch.ChannelType()

		f(receivedUser, receivedPass, receivedChannelType)
	}()

	return listener.Addr().String(), nil
}

func generateKeys() (string, string, string, error) {
	dir, err := ioutil.TempDir("", "pwd")
	if err != nil {
		return "", "", "", err
	}

	reader := rand.Reader
	bitSize := 2048

	key, err := rsa.GenerateKey(reader, bitSize)
	if err != nil {
		return "", "", "", err
	}

	privateFile, err := os.OpenFile(fmt.Sprintf("%s/id_rsa", dir), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return "", "", "", err
	}
	defer privateFile.Close()

	var privateKey = &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}

	err = pem.Encode(privateFile, privateKey)
	if err != nil {
		return "", "", "", err
	}

	publicFile, err := os.OpenFile(fmt.Sprintf("%s/id_rsa.pub", dir), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return "", "", "", err
	}
	defer publicFile.Close()

	asn1Bytes, err := asn1.Marshal(key.PublicKey)
	if err != nil {
		return "", "", "", err
	}

	var publicKey = &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: asn1Bytes,
	}

	err = pem.Encode(publicFile, publicKey)
	if err != nil {
		return "", "", "", err
	}
	return dir, privateFile.Name(), publicFile.Name(), nil
}

func getRouterUrl(scheme string, r *proxyRouter) string {
	chunks := strings.Split(r.ListenHttpAddress(), ":")
	return fmt.Sprintf("%s://localhost:%s", scheme, chunks[len(chunks)-1])
}

func routerLookup(protocol, domain string, r *proxyRouter) ([]string, error) {
	c := dns.Client{Net: protocol}
	m := dns.Msg{}

	m.SetQuestion(fmt.Sprintf("%s.", domain), dns.TypeA)
	var l string
	if protocol == "udp" {
		chunks := strings.Split(r.ListenDnsUdpAddress(), ":")
		l = fmt.Sprintf("127.0.0.1:%s", chunks[len(chunks)-1])
	} else if protocol == "tcp" {
		chunks := strings.Split(r.ListenDnsTcpAddress(), ":")
		l = fmt.Sprintf("127.0.0.1:%s", chunks[len(chunks)-1])
	}
	res, _, err := c.Exchange(&m, l)

	if err != nil {
		return nil, err
	}

	if len(res.Answer) == 0 {
		return nil, fmt.Errorf("Didn't receive any answer")
	}
	addrs := []string{}
	for _, a := range res.Answer {
		if b, ok := a.(*dns.A); ok {
			addrs = append(addrs, b.A.String())
		} else if b, ok := a.(*dns.AAAA); ok {
			addrs = append(addrs, b.AAAA.String())
		}
	}

	return addrs, nil
}

func TestProxy_TLS(t *testing.T) {
	dir, private, _, _ := generateKeys()
	defer os.RemoveAll(dir)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	const msg = "It works!"

	var receivedHost string
	var receivedProtocol Protocol

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, msg)
	}))
	defer ts.Close()

	r := NewRouter(func(protocol Protocol, host string) (*DirectorInfo, error) {
		receivedHost = host
		receivedProtocol = protocol
		u, _ := url.Parse(ts.URL)
		a, _ := net.ResolveTCPAddr("tcp", u.Host)
		return &DirectorInfo{Dst: a}, nil
	}, private)
	r.Listen(":0", ":0", ":0")
	defer r.Close()

	req, err := http.NewRequest("GET", getRouterUrl("https", r), nil)
	assert.Nil(t, err)

	resp, err := client.Do(req)
	assert.Nil(t, err)

	body, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, msg, string(body))
	assert.Equal(t, "localhost", receivedHost)
	assert.Equal(t, ProtocolHTTPS, receivedProtocol)
}

func TestProxy_Http(t *testing.T) {
	dir, private, _, _ := generateKeys()
	defer os.RemoveAll(dir)

	const msg = "It works!"

	var receivedHost string
	var receivedProtocol Protocol

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, msg)
	}))
	defer ts.Close()

	r := NewRouter(func(protocol Protocol, host string) (*DirectorInfo, error) {
		receivedHost = host
		receivedProtocol = protocol
		u, _ := url.Parse(ts.URL)
		a, _ := net.ResolveTCPAddr("tcp", u.Host)
		return &DirectorInfo{Dst: a}, nil
	}, private)
	r.Listen(":0", ":0", ":0")
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
	assert.Equal(t, ProtocolHTTP, receivedProtocol)
}

func TestProxy_WS(t *testing.T) {
	dir, private, _, _ := generateKeys()
	defer os.RemoveAll(dir)

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

	r := NewRouter(func(protocol Protocol, host string) (*DirectorInfo, error) {
		u, _ := url.Parse(ts.URL)
		a, _ := net.ResolveTCPAddr("tcp", u.Host)
		return &DirectorInfo{Dst: a}, nil
	}, private)
	r.Listen(":0", ":0", ":0")
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
	dir, private, _, _ := generateKeys()
	defer os.RemoveAll(dir)

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

	r := NewRouter(func(protocol Protocol, host string) (*DirectorInfo, error) {
		u, _ := url.Parse(ts.URL)
		a, _ := net.ResolveTCPAddr("tcp", u.Host)
		return &DirectorInfo{Dst: a}, nil
	}, private)
	r.Listen(":0", ":0", ":0")
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

func TestProxy_DNS_UDP(t *testing.T) {
	dir, private, _, _ := generateKeys()
	defer os.RemoveAll(dir)

	var receivedHost string
	var receivedProtocol Protocol

	r := NewRouter(func(protocol Protocol, host string) (*DirectorInfo, error) {
		receivedHost = host
		receivedProtocol = protocol
		if host == "10_0_0_1.foo.bar" {
			a, _ := net.ResolveTCPAddr("tcp", "10.0.0.1:0")
			return &DirectorInfo{Dst: a}, nil
		} else {
			return nil, fmt.Errorf("Not recognized")
		}
	}, private)
	r.Listen(":0", ":0", ":0")
	defer r.Close()

	ips, err := routerLookup("udp", "10_0_0_1.foo.bar", r)
	assert.Nil(t, err)
	assert.Equal(t, "10_0_0_1.foo.bar", receivedHost)
	assert.Equal(t, ProtocolDNS, receivedProtocol)
	assert.Equal(t, []string{"10.0.0.1"}, ips)

	ips, err = routerLookup("udp", "www.google.com", r)
	assert.Nil(t, err)
	assert.Equal(t, "www.google.com", receivedHost)
	assert.Equal(t, ProtocolDNS, receivedProtocol)

	expectedIps, err := net.LookupHost("www.google.com")
	assert.Nil(t, err)

	sort.Strings(expectedIps)
	sort.Strings(ips)
	assert.Equal(t, expectedIps, ips)

	ips, err = routerLookup("udp", "localhost", r)
	assert.Nil(t, err)
	assert.NotEqual(t, "localhost", receivedHost)
	assert.Equal(t, ProtocolDNS, receivedProtocol)
	assert.Equal(t, []string{"127.0.0.1"}, ips)
}

func TestProxy_DNS_TCP(t *testing.T) {
	dir, private, _, _ := generateKeys()
	defer os.RemoveAll(dir)

	var receivedHost string

	r := NewRouter(func(protocol Protocol, host string) (*DirectorInfo, error) {
		receivedHost = host
		if host == "10_0_0_1.foo.bar" {
			a, _ := net.ResolveTCPAddr("tcp", "10.0.0.1:0")
			return &DirectorInfo{Dst: a}, nil
		} else {
			return nil, fmt.Errorf("Not recognized")
		}
	}, private)
	r.Listen(":0", ":0", ":0")
	defer r.Close()

	ips, err := routerLookup("tcp", "10_0_0_1.foo.bar", r)
	assert.Nil(t, err)
	assert.Equal(t, "10_0_0_1.foo.bar", receivedHost)
	assert.Equal(t, []string{"10.0.0.1"}, ips)

	ips, err = routerLookup("tcp", "www.google.com", r)
	assert.Nil(t, err)
	assert.Equal(t, "www.google.com", receivedHost)

	expectedIps, err := net.LookupHost("www.google.com")
	assert.Nil(t, err)

	sort.Strings(expectedIps)
	sort.Strings(ips)
	assert.Equal(t, expectedIps, ips)

	ips, err = routerLookup("tcp", "localhost", r)
	assert.Nil(t, err)
	assert.NotEqual(t, "localhost", receivedHost)
	assert.Equal(t, []string{"127.0.0.1"}, ips)
}

func TestProxy_SSH(t *testing.T) {
	dir, private, _, _ := generateKeys()
	defer os.RemoveAll(dir)

	var receivedUser string
	var receivedPass string
	var receivedChannelType string
	var receivedHost string
	var receivedProtocol Protocol

	wg := sync.WaitGroup{}
	wg.Add(1)
	laddr, err := testSshServer(func(user, pass, ctype string) {
		receivedUser = user
		receivedPass = pass
		receivedChannelType = ctype
		wg.Done()
	})
	assert.Nil(t, err)

	r := NewRouter(func(protocol Protocol, host string) (*DirectorInfo, error) {
		receivedHost = host
		receivedProtocol = protocol
		if host == "10-0-0-1-aaaabbbb" {
			chunks := strings.Split(laddr, ":")
			a, _ := net.ResolveTCPAddr("tcp", fmt.Sprintf("127.0.0.1:%s", chunks[len(chunks)-1]))
			return &DirectorInfo{Dst: a, SSHUser: "root", SSHAuthMethods: []ssh.AuthMethod{ssh.Password("root")}}, nil
		} else {
			return nil, fmt.Errorf("Not recognized")
		}
	}, private)
	r.Listen(":0", ":0", ":0")
	defer r.Close()

	err = testSshClient("10-0-0-1-aaaabbbb", r)
	assert.Nil(t, err)

	wg.Wait()

	assert.Equal(t, "root", receivedUser)
	assert.Equal(t, "root", receivedPass)
	assert.Equal(t, "session", receivedChannelType)
	assert.Equal(t, "10-0-0-1-aaaabbbb", receivedHost)
	assert.Equal(t, ProtocolSSH, receivedProtocol)
}
