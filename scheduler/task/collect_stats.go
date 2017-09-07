package task

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	dockerTypes "github.com/docker/docker/api/types"
	units "github.com/docker/go-units"
	"github.com/play-with-docker/play-with-docker/docker"
	"github.com/play-with-docker/play-with-docker/event"
	"github.com/play-with-docker/play-with-docker/pwd/types"
)

type InstanceStats struct {
	Instance string `json:"instance"`
	Mem      string `json:"mem"`
	Cpu      string `json:"cpu"`
}

type collectStats struct {
	event   event.EventApi
	factory docker.FactoryApi
}

var CollectStatsEvent event.EventType

func init() {
	CollectStatsEvent = event.EventType("instance stats")
}

func (t *collectStats) Name() string {
	return "CollectStats"
}

func (t *collectStats) Run(ctx context.Context, instance *types.Instance) error {
	if instance.Type == "windows" {
		resp, err := http.Get(fmt.Sprintf("http://%s:222/stats", instance.IP))
		if err != nil {
			log.Printf("Could not get stats of windows instance with IP %s. Got: %v\n", instance.IP, err)
			return fmt.Errorf("Could not get stats of windows instance with IP %s. Got: %v\n", instance.IP, err)
		}
		if resp.StatusCode != 200 {
			log.Printf("Could not get stats of windows instance with IP %s. Got status code: %d\n", instance.IP, resp.StatusCode)
			return fmt.Errorf("Could not get stats of windows instance with IP %s. Got status code: %d\n", instance.IP, resp.StatusCode)
		}
		var info map[string]float64
		err = json.NewDecoder(resp.Body).Decode(&info)
		if err != nil {
			log.Printf("Could not get stats of windows instance with IP %s. Got: %v\n", instance.IP, err)
			return fmt.Errorf("Could not get stats of windows instance with IP %s. Got: %v\n", instance.IP, err)
		}
		stats := InstanceStats{Instance: instance.Name}

		stats.Mem = fmt.Sprintf("%.2f%% (%s / %s)", ((info["mem_used"] / info["mem_total"]) * 100), units.BytesSize(info["mem_used"]), units.BytesSize(info["mem_total"]))
		stats.Cpu = fmt.Sprintf("%.2f%%", info["cpu"]*100)
		t.event.Emit(CollectStatsEvent, instance.SessionId, stats)
		return nil
	}
	dockerClient, err := t.factory.GetForSession(instance.SessionId)
	if err != nil {
		log.Println(err)
		return err
	}
	reader, err := dockerClient.GetContainerStats(instance.Name)
	if err != nil {
		log.Println("Error while trying to collect instance stats", err)
		return err
	}
	dec := json.NewDecoder(reader)
	var v *dockerTypes.StatsJSON
	e := dec.Decode(&v)
	if e != nil {
		log.Println("Error while trying to collect instance stats", e)
		return err
	}
	stats := InstanceStats{Instance: instance.Name}
	// Memory
	var memPercent float64 = 0
	if v.MemoryStats.Limit != 0 {
		memPercent = float64(v.MemoryStats.Usage) / float64(v.MemoryStats.Limit) * 100.0
	}
	mem := float64(v.MemoryStats.Usage)
	memLimit := float64(v.MemoryStats.Limit)

	stats.Mem = fmt.Sprintf("%.2f%% (%s / %s)", memPercent, units.BytesSize(mem), units.BytesSize(memLimit))

	// cpu
	previousCPU := v.PreCPUStats.CPUUsage.TotalUsage
	previousSystem := v.PreCPUStats.SystemUsage
	cpuPercent := calculateCPUPercentUnix(previousCPU, previousSystem, v)
	stats.Cpu = fmt.Sprintf("%.2f%%", cpuPercent)

	t.event.Emit(CollectStatsEvent, instance.SessionId, stats)
	return nil
}

func NewCollectStats(e event.EventApi, f docker.FactoryApi) *collectStats {
	return &collectStats{event: e, factory: f}
}

func calculateCPUPercentUnix(previousCPU, previousSystem uint64, v *dockerTypes.StatsJSON) float64 {
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
