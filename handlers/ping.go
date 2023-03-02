package handlers

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/docker/docker/client"
	"github.com/shirou/gopsutil/load"
	"github.com/thebsdbox/play-with-docker/config"
)

func Ping(rw http.ResponseWriter, req *http.Request) {
	defer latencyHistogramVec.WithLabelValues("ping").Observe(float64(time.Since(time.Now()).Nanoseconds()) / 1000000)
	// Get system load average of the last 5 minutes and compare it against a threashold.

	c, err := client.NewClientWithOpts()

	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := c.Info(ctx); err != nil && err == context.DeadlineExceeded {
		log.Printf("Docker info took to long to respond\n")
		rw.WriteHeader(http.StatusGatewayTimeout)
		return
	}

	a, err := load.Avg()
	if err != nil {
		log.Println("Cannot get system load average!", err)
	} else {
		if a.Load5 > config.MaxLoadAvg {
			log.Printf("System load average is too high [%f]\n", a.Load5)
			rw.WriteHeader(http.StatusInsufficientStorage)
		}
	}
}
