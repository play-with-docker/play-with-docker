package handlers

import (
	"log"
	"net/http"

	"github.com/play-with-docker/play-with-docker/config"
	"github.com/shirou/gopsutil/load"
)

func Ping(rw http.ResponseWriter, req *http.Request) {
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
}
