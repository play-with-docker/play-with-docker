package pwd

import (
	"encoding/json"
	"fmt"
	"log"

	dockerTypes "github.com/docker/docker/api/types"
	units "github.com/docker/go-units"
	"github.com/play-with-docker/play-with-docker/docker"
	"github.com/play-with-docker/play-with-docker/pwd/types"
)

type collectStatsTask struct {
	mem        float64
	memLimit   float64
	memPercent float64

	cpuPercent     float64
	previousCPU    uint64
	previousSystem uint64

	docker docker.DockerApi
}

func (c collectStatsTask) Run(i *types.Instance) error {
	reader, err := c.docker.GetContainerStats(i.Name)
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
	// Memory
	if v.MemoryStats.Limit != 0 {
		c.memPercent = float64(v.MemoryStats.Usage) / float64(v.MemoryStats.Limit) * 100.0
	}
	c.mem = float64(v.MemoryStats.Usage)
	c.memLimit = float64(v.MemoryStats.Limit)

	i.Mem = fmt.Sprintf("%.2f%% (%s / %s)", c.memPercent, units.BytesSize(c.mem), units.BytesSize(c.memLimit))

	// cpu
	c.previousCPU = v.PreCPUStats.CPUUsage.TotalUsage
	c.previousSystem = v.PreCPUStats.SystemUsage
	c.cpuPercent = calculateCPUPercentUnix(c.previousCPU, c.previousSystem, v)
	i.Cpu = fmt.Sprintf("%.2f%%", c.cpuPercent)

	return nil
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
