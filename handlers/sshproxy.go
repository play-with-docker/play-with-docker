package handlers

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
)

var sshConfig = &ssh.ServerConfig{
	PublicKeyCallback: func(c ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
		user := c.User()
		chunks := strings.Split(user, "-")
		ip := strings.Join(chunks[:4], ".")
		sessionPrefix := chunks[4]

		log.Println(ip, sessionPrefix)

		return nil, nil
	},
}

func ListenSSHProxy(laddr string) {
	privateBytes, err := ioutil.ReadFile("/etc/ssh/ssh_host_rsa_key")
	if err != nil {
		log.Fatal("Failed to load private key: ", err)
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		log.Fatal("Failed to parse private key: ", err)
	}

	sshConfig.AddHostKey(private)

	listener, err := net.Listen("tcp", laddr)
	if err != nil {
		log.Fatal("failed to listen for connection: ", err)
	}
	for {
		nConn, err := listener.Accept()
		if err != nil {
			log.Fatal("failed to accept incoming connection: ", err)
		}

		go handle(nConn)
	}
}

func handle(c net.Conn) {
	sshCon, chans, reqs, err := ssh.NewServerConn(c, sshConfig)
	if err != nil {
		c.Close()
		return
	}

	user := sshCon.User()
	chunks := strings.Split(user, "-")
	ip := strings.Join(chunks[:4], ".")
	sessionPrefix := chunks[4]

	i := core.InstanceFindByIPAndSession(sessionPrefix, ip)
	if i == nil {
		log.Printf("Couldn't find instance with ip [%s] in session [%s]\n", ip, sessionPrefix)
		c.Close()
		return
	}

	// The incoming Request channel must be serviced.
	go ssh.DiscardRequests(reqs)

	newChannel := <-chans
	if newChannel == nil {
		sshCon.Close()
		return
	}

	if newChannel.ChannelType() != "session" {
		newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
		return
	}

	channel, requests, err := newChannel.Accept()
	if err != nil {
		log.Fatalf("Could not accept channel: %v", err)
	}

	stderr := channel.Stderr()

	fmt.Fprintf(stderr, "Connecting to %s\r\n", ip)

	clientConfig := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.Password("root"),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", ip), clientConfig)
	if err != nil {
		fmt.Fprintf(stderr, "Connect failed: %v\r\n", err)
		channel.Close()
		return
	}

	go func() {
		for newChannel = range chans {
			if newChannel == nil {
				return
			}

			channel2, reqs2, err := client.OpenChannel(newChannel.ChannelType(), newChannel.ExtraData())
			if err != nil {
				x, ok := err.(*ssh.OpenChannelError)
				if ok {
					newChannel.Reject(x.Reason, x.Message)
				} else {
					newChannel.Reject(ssh.Prohibited, "remote server denied channel request")
				}
				continue
			}

			channel, reqs, err := newChannel.Accept()
			if err != nil {
				channel2.Close()
				continue
			}
			go proxy(reqs, reqs2, channel, channel2)
		}
	}()

	// Forward the session channel
	channel2, reqs2, err := client.OpenChannel("session", []byte{})
	if err != nil {
		fmt.Fprintf(stderr, "Remote session setup failed: %v\r\n", err)
		channel.Close()
		return
	}

	maskedReqs := make(chan *ssh.Request, 1)
	go func() {
		for req := range requests {
			if req.Type == "auth-agent-req@openssh.com" {
				continue
			}
			maskedReqs <- req
		}
	}()
	proxy(maskedReqs, reqs2, channel, channel2)
}

func proxy(reqs1, reqs2 <-chan *ssh.Request, channel1, channel2 ssh.Channel) {
	var closer sync.Once
	closeFunc := func() {
		channel1.Close()
		channel2.Close()
	}

	defer closer.Do(closeFunc)

	closerChan := make(chan bool, 1)

	go func() {
		io.Copy(channel1, channel2)
		closerChan <- true
	}()

	go func() {
		io.Copy(channel2, channel1)
		closerChan <- true
	}()

	for {
		select {
		case req := <-reqs1:
			if req == nil {
				return
			}
			b, err := channel2.SendRequest(req.Type, req.WantReply, req.Payload)
			if err != nil {
				return
			}
			req.Reply(b, nil)

		case req := <-reqs2:
			if req == nil {
				return
			}
			b, err := channel1.SendRequest(req.Type, req.WantReply, req.Payload)
			if err != nil {
				return
			}
			req.Reply(b, nil)
		case <-closerChan:
			return
		}
	}
}
