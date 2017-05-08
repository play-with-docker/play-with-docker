package services

import (
	"encoding/gob"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api"
	"github.com/docker/docker/client"
	"github.com/googollee/go-socket.io"
	"github.com/play-with-docker/play-with-docker/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/twinj/uuid"
)

var (
	sessionsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "sessions",
		Help: "Sessions",
	})
	clientsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "clients",
		Help: "Clients",
	})
	instancesGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "instances",
		Help: "Instances",
	})
)

func init() {
	prometheus.MustRegister(sessionsGauge)
	prometheus.MustRegister(clientsGauge)
	prometheus.MustRegister(instancesGauge)
}

var wsServer *socketio.Server

type Session struct {
	rw           sync.Mutex
	Id           string               `json:"id"`
	Instances    map[string]*Instance `json:"instances"`
	clients      []*Client            `json:"-"`
	CreatedAt    time.Time            `json:"created_at"`
	ExpiresAt    time.Time            `json:"expires_at"`
	scheduled    bool                 `json:"-"`
	ticker       *time.Ticker         `json:"-"`
	PwdIpAddress string               `json:"pwd_ip_address"`
}

func (s *Session) Lock() {
	s.rw.Lock()
}

func (s *Session) Unlock() {
	s.rw.Unlock()
}

func (s *Session) GetSmallestViewPort() ViewPort {
	minRows := s.clients[0].ViewPort.Rows
	minCols := s.clients[0].ViewPort.Cols

	for _, c := range s.clients {
		minRows = uint(math.Min(float64(minRows), float64(c.ViewPort.Rows)))
		minCols = uint(math.Min(float64(minCols), float64(c.ViewPort.Cols)))
	}

	return ViewPort{Rows: minRows, Cols: minCols}
}

func (s *Session) AddNewClient(c *Client) {
	s.clients = append(s.clients, c)
	setGauges()
}

func (s *Session) SchedulePeriodicTasks() {
	if s.scheduled {
		return
	}

	go func() {
		s.scheduled = true

		s.ticker = time.NewTicker(1 * time.Second)
		for range s.ticker.C {
			var wg = sync.WaitGroup{}
			wg.Add(len(s.Instances))
			for _, ins := range s.Instances {
				var i *Instance = ins
				if i.dockerClient == nil {
					// Need to create client to the DinD docker daemon

					transport := &http.Transport{
						DialContext: (&net.Dialer{
							Timeout:   1 * time.Second,
							KeepAlive: 30 * time.Second,
						}).DialContext}
					cli := &http.Client{
						Transport: transport,
					}
					c, err := client.NewClient(fmt.Sprintf("http://%s:2375", i.IP), api.DefaultVersion, cli, nil)
					if err != nil {
						log.Println("Could not connect to DinD docker daemon", err)
					} else {
						i.dockerClient = c
					}
				}
				go func() {
					defer wg.Done()
					for _, t := range periodicTasks {
						err := t.Run(i)
						if err != nil {
							if strings.Contains(err.Error(), "No such container") {
								log.Printf("Container for instance [%s] doesn't exist any more. Deleting from session.\n", i.IP)
								DeleteInstance(i.session, i)
							} else {
								log.Println(err)
							}
							break
						}
					}
				}()
			}
			wg.Wait()
			// broadcast all information
			for _, ins := range s.Instances {
				ins.Ports = UInt16Slice(ins.tempPorts)
				sort.Sort(ins.Ports)
				ins.tempPorts = []uint16{}

				wsServer.BroadcastTo(ins.session.Id, "instance stats", ins.Name, ins.Mem, ins.Cpu, ins.IsManager, ins.Ports)
			}
		}
	}()
}

var sessions map[string]*Session

func init() {
	sessions = make(map[string]*Session)
}

func CreateWSServer() *socketio.Server {
	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}
	wsServer = server
	return server
}

func CloseSessionAfter(s *Session, d time.Duration) {
	time.AfterFunc(d, func() {
		CloseSession(s)
	})
}

func CloseSession(s *Session) error {
	s.rw.Lock()
	defer s.rw.Unlock()

	if s.ticker != nil {
		s.ticker.Stop()
	}
	wsServer.BroadcastTo(s.Id, "session end")
	for _, c := range s.clients {
		c.so.Emit("disconnect")
	}
	log.Printf("Starting clean up of session [%s]\n", s.Id)
	for _, i := range s.Instances {
		err := DeleteInstance(s, i)
		if err != nil {
			log.Println(err)
			return err
		}
	}
	// Disconnect PWD daemon from the network
	if err := DisconnectNetwork("pwd", s.Id); err != nil {
		if !strings.Contains(err.Error(), "is not connected to the network") {
			log.Println("ERROR NETWORKING")
			return err
		}
	}
	log.Printf("Disconnected pwd from network [%s]\n", s.Id)
	if err := DeleteNetwork(s.Id); err != nil {
		if !strings.Contains(err.Error(), "not found") {
			log.Println(err)
			return err
		}
	}
	delete(sessions, s.Id)

	// We store sessions as soon as we delete one
	if err := saveSessionsToDisk(); err != nil {
		return err
	}
	setGauges()
	log.Printf("Cleaned up session [%s]\n", s.Id)
	return nil
}

var defaultDuration = 4 * time.Hour

func GetDuration(reqDur string) time.Duration {
	if reqDur != "" {
		if dur, err := time.ParseDuration(reqDur); err == nil && dur <= defaultDuration {
			return dur
		}
		return defaultDuration
	}

	envDur := os.Getenv("EXPIRY")
	if dur, err := time.ParseDuration(envDur); err == nil {
		return dur
	}

	return defaultDuration
}

func NewSession(duration time.Duration) (*Session, error) {
	s := &Session{}
	s.Id = uuid.NewV4().String()
	s.Instances = map[string]*Instance{}
	s.CreatedAt = time.Now()
	s.ExpiresAt = s.CreatedAt.Add(duration)
	log.Printf("NewSession id=[%s]\n", s.Id)

	// Schedule cleanup of the session
	CloseSessionAfter(s, duration)

	if err := CreateNetwork(s.Id); err != nil {
		log.Println("ERROR NETWORKING")
		return nil, err
	}
	log.Printf("Network [%s] created for session [%s]\n", s.Id, s.Id)

	// Connect PWD daemon to the new network
	ip, err := ConnectNetwork(config.PWDContainerName, s.Id, "")
	if err != nil {
		log.Println("ERROR NETWORKING")
		return nil, err
	}
	s.PwdIpAddress = ip
	log.Printf("Connected %s to network [%s]\n", config.PWDContainerName, s.Id)

	// Schedule peridic tasks execution
	s.SchedulePeriodicTasks()

	sessions[s.Id] = s

	// We store sessions as soon as we create one so we don't delete new sessions on an api restart
	if err := saveSessionsToDisk(); err != nil {
		return nil, err
	}

	setGauges()
	return s, nil
}

func GetSession(sessionId string) *Session {
	s := sessions[sessionId]
	if s != nil {
		for _, instance := range s.Instances {
			if !instance.IsConnected() {
				instance.SetSession(s)
				go instance.Attach()
			}
		}

	}
	return s
}

func setGauges() {
	var ins float64
	var cli float64

	for _, s := range sessions {
		ins += float64(len(s.Instances))
		cli += float64(len(s.clients))
	}

	clientsGauge.Set(cli)
	instancesGauge.Set(ins)
	sessionsGauge.Set(float64(len(sessions)))
}

func LoadSessionsFromDisk() error {
	file, err := os.Open(config.SessionsFile)
	if err == nil {
		decoder := gob.NewDecoder(file)
		err = decoder.Decode(&sessions)

		if err != nil {
			return err
		}

		// schedule session expiration
		for _, s := range sessions {
			timeLeft := s.ExpiresAt.Sub(time.Now())
			CloseSessionAfter(s, timeLeft)

			// start collecting stats for every instance
			for _, i := range s.Instances {
				// wire the session back to the instance
				i.session = s

				if i.ServerCert != nil && i.ServerKey != nil {
					_, err := i.SetCertificate(i.ServerCert, i.ServerKey)
					if err != nil {
						log.Println(err)
						return err
					}
				}
			}

			// Connect PWD daemon to the new network
			if s.PwdIpAddress == "" {
				log.Fatal("Cannot load stored sessions as they don't have the pwd ip address stored with them")
			}
			if _, err := ConnectNetwork(config.PWDContainerName, s.Id, s.PwdIpAddress); err != nil {
				if strings.Contains(err.Error(), "Could not attach to network") {
					log.Printf("Network for session [%s] doesn't exist. Removing all instances and session.", s.Id)
					CloseSession(s)
				} else {
					log.Println("ERROR NETWORKING", err)
					return err
				}
			} else {
				log.Printf("Connected %s to network [%s]\n", config.PWDContainerName, s.Id)

				// Schedule peridic tasks execution
				s.SchedulePeriodicTasks()
			}
		}
	}
	file.Close()
	setGauges()
	return err
}

func saveSessionsToDisk() error {
	rw.Lock()
	defer rw.Unlock()
	file, err := os.Create(config.SessionsFile)
	if err == nil {
		encoder := gob.NewEncoder(file)
		err = encoder.Encode(&sessions)
	}
	file.Close()
	return err
}
