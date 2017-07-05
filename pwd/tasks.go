package pwd

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/play-with-docker/play-with-docker/docker"
)

type periodicTask interface {
	Run(i *Instance) error
}

type SchedulerApi interface {
	Schedule(session *Session)
	Unschedule(session *Session)
}

type scheduler struct {
	broadcast     BroadcastApi
	periodicTasks []periodicTask
}

func (sch *scheduler) Schedule(s *Session) {
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
				if i.k8s == nil {
					c := rest.Config{Host: fmt.Sprintf("%s:6443", i.IP), TLSClientConfig: rest.TLSClientConfig{Insecure: true}, BearerToken: "system:admin/system:masters"}
					client, err := kubernetes.NewForConfig(&c)
					if err != nil {
						log.Println("Could not connect to k8s server api", err)
					}
					i.k8s = client
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
				ins.Ports = UInt16Slice(ins.tempPorts)
				sort.Sort(ins.Ports)
				ins.tempPorts = []uint16{}

				sch.broadcast.BroadcastTo(ins.session.Id, "instance stats", ins.Name, ins.Mem, ins.Cpu, ins.IsManager, ins.Ports)
			}
		}
	}()
}

func (sch *scheduler) Unschedule(s *Session) {
}

func NewScheduler(b BroadcastApi, d docker.DockerApi) *scheduler {
	s := &scheduler{broadcast: b}
	s.periodicTasks = []periodicTask{&collectStatsTask{docker: d}, &checkK8sClusterStatusTask{}, &checkK8sClusterExposedPortsTask{}}
	return s
}
