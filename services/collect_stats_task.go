package services

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/docker/docker/api/types"
	units "github.com/docker/go-units"
)

type collectStatsTask struct {
	mem        float64
	memLimit   float64
	memPercent float64
	v          *types.StatsJSON

	cpuPercent     float64
	previousCPU    uint64
	previousSystem uint64
}

func (c *collectStatsTask) Run(i *Instance) {
	reader, err := GetContainerStats(i.Name)
	if err != nil {
		log.Println("Error while trying to collect instance stats", err)
		return
	}
	dec := json.NewDecoder(reader)
	e := dec.Decode(&c.v)
	if e != nil {
		log.Println("Error while trying to collect instance stats", e)
		return
	}
	// Memory
	if c.v.MemoryStats.Limit != 0 {
		c.memPercent = float64(c.v.MemoryStats.Usage) / float64(c.v.MemoryStats.Limit) * 100.0
	}
	c.mem = float64(c.v.MemoryStats.Usage)
	c.memLimit = float64(c.v.MemoryStats.Limit)

	i.Mem = fmt.Sprintf("%.2f%% (%s / %s)", c.memPercent, units.BytesSize(c.mem), units.BytesSize(c.memLimit))

	// cpu
	c.previousCPU = c.v.PreCPUStats.CPUUsage.TotalUsage
	c.previousSystem = c.v.PreCPUStats.SystemUsage
	c.cpuPercent = calculateCPUPercentUnix(c.previousCPU, c.previousSystem, c.v)
	i.Cpu = fmt.Sprintf("%.2f%%", c.cpuPercent)
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
