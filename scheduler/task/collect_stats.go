package task

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	dockerTypes "docker.io/go-docker/api/types"
	units "github.com/docker/go-units"
	lru "github.com/hashicorp/golang-lru"
	"github.com/play-with-docker/play-with-docker/docker"
	"github.com/play-with-docker/play-with-docker/event"
	"github.com/play-with-docker/play-with-docker/pwd/types"
	"github.com/play-with-docker/play-with-docker/router"
	"github.com/play-with-docker/play-with-docker/storage"
)

type InstanceStats struct {
	Instance string `json:"instance"`
	Mem      string `json:"mem"`
	Cpu      string `json:"cpu"`
}

type collectStats struct {
	event   event.EventApi
	factory docker.FactoryApi
	cli     *http.Client
	cache   *lru.Cache
	storage storage.StorageApi
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
		host := router.EncodeHost(instance.SessionId, instance.IP, router.HostOpts{EncodedPort: 222})
		req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/stats", host), nil)
		if err != nil {
			log.Printf("Could not create request to get stats of windows instance with IP %s. Got: %v\n", instance.IP, err)
			return fmt.Errorf("Could not create request to get stats of windows instance with IP %s. Got: %v\n", instance.IP, err)
		}
		req.Header.Set("X-Proxy-Host", instance.SessionHost)
		resp, err := t.cli.Do(req)
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
	var session *types.Session
	if sess, found := t.cache.Get(instance.SessionId); !found {
		s, err := t.storage.SessionGet(instance.SessionId)
		if err != nil {
			return err
		}
		t.cache.Add(s.Id, s)
		session = s
	} else {
		session = sess.(*types.Session)
	}
	dockerClient, err := t.factory.GetForSession(session)
	if err != nil {
		log.Println(err)
		return err
	}
	reader, err := dockerClient.ContainerStats(instance.Name)
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

func proxyHost(r *http.Request) (*url.URL, error) {
	if r.Header.Get("X-Proxy-Host") == "" {
		return nil, nil
	}
	u := new(url.URL)
	*u = *r.URL
	u.Host = fmt.Sprintf("%s:8443", r.Header.Get("X-Proxy-Host"))
	return u, nil
}

func NewCollectStats(e event.EventApi, f docker.FactoryApi, s storage.StorageApi) *collectStats {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   1 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConnsPerHost: 5,
		Proxy:               proxyHost,
	}
	cli := &http.Client{
		Transport: transport,
	}
	c, _ := lru.New(5000)
	return &collectStats{event: e, factory: f, cli: cli, cache: c, storage: s}
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
