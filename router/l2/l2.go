package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"golang.org/x/crypto/ssh"

	client "docker.io/go-docker"
	"docker.io/go-docker/api/types"
	"docker.io/go-docker/api/types/filters"
	"docker.io/go-docker/api/types/network"
	"github.com/gorilla/mux"
	"github.com/play-with-docker/play-with-docker/config"
	"github.com/play-with-docker/play-with-docker/router"
	"github.com/shirou/gopsutil/load"
	"github.com/urfave/negroni"
)

func director(protocol router.Protocol, host string) (*router.DirectorInfo, error) {
	info, err := router.DecodeHost(host)
	if err != nil {
		return nil, err
	}

	port := info.Port

	if info.EncodedPort > 0 {
		port = info.EncodedPort
	}

	i := router.DirectorInfo{}
	if port == 0 {
		if protocol == router.ProtocolHTTP {
			port = 80
		} else if protocol == router.ProtocolHTTPS {
			port = 443
		} else if protocol == router.ProtocolSSH {
			port = 22
			i.SSHUser = "root"
			i.SSHAuthMethods = []ssh.AuthMethod{ssh.Password("root")}
		} else if protocol == router.ProtocolDNS {
			port = 53
		}
	}

	t, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", info.InstanceIP, port))
	if err != nil {
		return nil, err
	}
	i.Dst = t
	return &i, nil
}

func connectNetworks() error {
	ctx := context.Background()
	c, err := client.NewEnvClient()
	if err != nil {
		log.Fatal(err)
	}

	defer c.Close()

	f, err := os.Open(config.SessionsFile)
	if err != nil {
		return err
	}
	defer f.Close()

	networks := map[string]*network.EndpointSettings{}

	err = json.NewDecoder(f).Decode(&networks)
	if err != nil {
		return err
	}

	for netId, opts := range networks {
		settings := &network.EndpointSettings{}
		settings.IPAddress = opts.IPAddress
		log.Printf("Connected to network [%s] with ip [%s]\n", netId, opts.IPAddress)
		c.NetworkConnect(ctx, netId, config.PWDContainerName, settings)
	}

	return nil
}

func monitorNetworks() {
	c, err := client.NewEnvClient()
	if err != nil {
		log.Fatal(err)
	}

	defer c.Close()

	ctx := context.Background()

	args := filters.NewArgs()

	cmsg, _ := c.Events(ctx, types.EventsOptions{Filters: args})
	for {
		select {
		case m := <-cmsg:
			if m.Type == "network" {
				// Router has been connected to a new network. Let's get all connections and store them in case of restart.
				container, err := c.ContainerInspect(ctx, config.PWDContainerName)
				if err != nil {
					log.Println(err)
					return
				}

				f, err := os.Create(config.SessionsFile)
				if err != nil {
					log.Println(err)
					return
				}
				err = json.NewEncoder(f).Encode(container.NetworkSettings.Networks)
				if err != nil {
					log.Println(err)
					return
				}
				log.Println("Saved networks")
			}
		}
	}
}

func main() {
	config.ParseFlags()

	err := connectNetworks()
	if err != nil && !os.IsNotExist(err) {
		log.Fatal("connect networks:", err)
	}
	go monitorNetworks()

	ro := mux.NewRouter()
	ro.HandleFunc("/ping", ping).Methods("GET")
	n := negroni.Classic()
	n.UseHandler(ro)

	httpServer := http.Server{
		Addr:              "0.0.0.0:8080",
		Handler:           n,
		IdleTimeout:       30 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go httpServer.ListenAndServe()

	r := router.NewRouter(director, config.SSHKeyPath)
	r.ListenAndWait(":443", ":53", ":22")
	defer r.Close()
}

func ping(rw http.ResponseWriter, req *http.Request) {
	// Get system load average of the last 5 minutes and compare it against a threashold.

	a, err := load.Avg()
	if err != nil {
		log.Println("Cannot get system load average!", err)
	} else {
		if a.Load5 > config.MaxLoadAvg {
			log.Printf("System load average is too high [%f]\n", a.Load5)
			rw.WriteHeader(http.StatusInsufficientStorage)
		}
	}

	fmt.Fprintf(rw, `{"ip": "%s"}`, config.L2RouterIP)
}
