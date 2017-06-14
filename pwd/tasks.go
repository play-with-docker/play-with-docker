package pwd

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/tlsconfig"
	"github.com/play-with-docker/play-with-docker/docker"
	"github.com/play-with-docker/play-with-docker/pwd/types"
)

type periodicTask interface {
	Run(i *types.Instance) error
}

type SchedulerApi interface {
	Schedule(session *types.Session)
	Unschedule(session *types.Session)
}

type scheduler struct {
	broadcast     BroadcastApi
	periodicTasks []periodicTask
}

func (sch *scheduler) Schedule(s *types.Session) {
	if s.IsPrepared() {
		return
	}

	go func() {
		t := time.NewTicker(1 * time.Second)
		s.SetTicker(t)
		for range t.C {
			var wg = sync.WaitGroup{}
			wg.Add(len(s.Instances))
			for _, ins := range s.Instances {
				var i *types.Instance = ins
				if i.Docker == nil && i.IsDockerHost {
					// Need to create client to the DinD docker daemon

					// We check if the client needs to use TLS
					var tlsConfig *tls.Config
					if len(i.Cert) > 0 && len(i.Key) > 0 {
						tlsConfig = tlsconfig.ClientDefault()
						tlsConfig.InsecureSkipVerify = true
						tlsCert, err := tls.X509KeyPair(i.Cert, i.Key)
						if err != nil {
							log.Println("Could not load X509 key pair: %v. Make sure the key is not encrypted", err)
							continue
						}
						tlsConfig.Certificates = []tls.Certificate{tlsCert}
					}

					transport := &http.Transport{
						DialContext: (&net.Dialer{
							Timeout:   1 * time.Second,
							KeepAlive: 30 * time.Second,
						}).DialContext}
					if tlsConfig != nil {
						transport.TLSClientConfig = tlsConfig
					}
					cli := &http.Client{
						Transport: transport,
					}
					c, err := client.NewClient(fmt.Sprintf("http://%s:2375", i.IP), api.DefaultVersion, cli, nil)
					if err != nil {
						log.Println("Could not connect to DinD docker daemon", err)
					} else {
						i.Docker = docker.NewDocker(c)
					}
				}
				go func() {
					defer wg.Done()
					for _, t := range sch.periodicTasks {
						err := t.Run(i)
						if err != nil {
							if strings.Contains(err.Error(), "No such container") {
								log.Printf("Container for instance [%s] doesn't exist any more.\n", i.IP)
								//DeleteInstance(i.session, i)
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
				ins.Ports = types.UInt16Slice(ins.GetUsedPorts())
				sort.Sort(ins.Ports)
				ins.CleanUsedPorts()

				sch.broadcast.BroadcastTo(ins.Session.Id, "instance stats", ins.Name, ins.Mem, ins.Cpu, ins.IsManager, ins.Ports)
			}
		}
	}()
}

func (sch *scheduler) Unschedule(s *types.Session) {
}

func NewScheduler(b BroadcastApi, d docker.DockerApi) *scheduler {
	s := &scheduler{broadcast: b}
	s.periodicTasks = []periodicTask{&collectStatsTask{docker: d}, &checkSwarmStatusTask{}, &checkUsedPortsTask{}, &checkSwarmUsedPortsTask{}}
	return s
}
