package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"golang.org/x/text/encoding"

	"github.com/docker/docker/api/types"
	units "github.com/docker/go-units"
)

var rw sync.Mutex

type Instance struct {
	session     *Session                `json:"-"`
	Name        string                  `json:"name"`
	Hostname    string                  `json:"hostname"`
	IP          string                  `json:"ip"`
	conn        *types.HijackedResponse `json:"-"`
	ctx         context.Context         `json:"-"`
	statsReader io.ReadCloser           `json:"-"`
}

func (i *Instance) IsConnected() bool {
	return i.conn != nil

}

func (i *Instance) SetSession(s *Session) {
	i.session = s
}

var dindImage string
var defaultDindImageName string

func init() {
	dindImage = getDindImageName()
}

func getDindImageName() string {
	dindImage := os.Getenv("DIND_IMAGE")
	defaultDindImageName = "franela/pwd-1.12.3-experimental-dind"
	if len(dindImage) == 0 {
		dindImage = defaultDindImageName
	}
	return dindImage
}

func NewInstance(session *Session) (*Instance, error) {
	log.Printf("NewInstance - using image: [%s]\n", dindImage)
	instance, err := CreateInstance(session.Id, dindImage)
	if err != nil {
		return nil, err
	}
	instance.session = session

	if session.Instances == nil {
		session.Instances = make(map[string]*Instance)
	}
	session.Instances[instance.Name] = instance

	go instance.Attach()

	err = saveSessionsToDisk()
	if err != nil {
		return nil, err
	}

	wsServer.BroadcastTo(session.Id, "new instance", instance.Name, instance.IP, instance.Hostname)

	// Start collecting stats
	go instance.CollectStats()

	return instance, nil
}

type sessionWriter struct {
	instance *Instance
}

func (s *sessionWriter) Write(p []byte) (n int, err error) {
	wsServer.BroadcastTo(s.instance.session.Id, "terminal out", s.instance.Name, string(p))
	return len(p), nil
}

func (o *Instance) CollectStats() {
	reader, err := GetContainerStats(o.Hostname)
	if err != nil {
		log.Println("Error while trying to collect instance stats", err)
		return
	}
	o.statsReader = reader
	dec := json.NewDecoder(o.statsReader)
	var (
		mem          = 0.0
		memLimit     = 0.0
		memPercent   = 0.0
		v            *types.StatsJSON
		memFormatted = ""

		cpuPercent     = 0.0
		previousCPU    uint64
		previousSystem uint64
		cpuFormatted   = ""
	)
	for {
		e := dec.Decode(&v)
		if e != nil {
			break
		}

		// Memory
		if v.MemoryStats.Limit != 0 {
			memPercent = float64(v.MemoryStats.Usage) / float64(v.MemoryStats.Limit) * 100.0
		}
		mem = float64(v.MemoryStats.Usage)
		memLimit = float64(v.MemoryStats.Limit)

		memFormatted = fmt.Sprintf("%.2f%% (%s / %s)", memPercent, units.BytesSize(mem), units.BytesSize(memLimit))

		// cpu
		previousCPU = v.PreCPUStats.CPUUsage.TotalUsage
		previousSystem = v.PreCPUStats.SystemUsage
		cpuPercent = calculateCPUPercentUnix(previousCPU, previousSystem, v)
		cpuFormatted = fmt.Sprintf("%.2f%%", cpuPercent)

		wsServer.BroadcastTo(o.session.Id, "instance stats", o.Name, memFormatted, cpuFormatted)
	}

}

func calculateCPUPercentUnix(previousCPU, previousSystem uint64, v *types.StatsJSON) float64 {
	var (
		cpuPercent = 0.0
		// calculate the change for the cpu usage of the container in between readings
		cpuDelta = float64(v.CPUStats.CPUUsage.TotalUsage) - float64(previousCPU)
		// calculate the change for the entire system between readings
		systemDelta = float64(v.CPUStats.SystemUsage) - float64(previousSystem)
	)

	if systemDelta > 0.0 && cpuDelta > 0.0 {
		cpuPercent = (cpuDelta / systemDelta) * float64(len(v.CPUStats.CPUUsage.PercpuUsage)) * 100.0
	}
	return cpuPercent
}

func (i *Instance) ResizeTerminal(cols, rows uint) error {
	return ResizeConnection(i.Name, cols, rows)
}

func (i *Instance) Attach() {
	i.ctx = context.Background()
	conn, err := CreateAttachConnection(i.Name, i.ctx)

	if err != nil {
		return
	}

	i.conn = conn

	go func() {
		encoder := encoding.Replacement.NewEncoder()
		sw := &sessionWriter{instance: i}
		io.Copy(encoder.Writer(sw), conn.Reader)
	}()

	select {
	case <-i.ctx.Done():
	}
}
func GetInstance(session *Session, name string) *Instance {
	//TODO: Use redis
	return session.Instances[name]
}
func DeleteInstance(session *Session, instance *Instance) error {
	// stop collecting stats
	if instance.statsReader != nil {
		instance.statsReader.Close()
	}

	//TODO: Use redis
	delete(session.Instances, instance.Name)
	err := DeleteContainer(instance.Name)

	wsServer.BroadcastTo(session.Id, "delete instance", instance.Name)

	return err
}
