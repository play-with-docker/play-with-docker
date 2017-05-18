package handlers

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/franela/play-with-docker/config"
	"github.com/franela/play-with-docker/services"
	"github.com/shirou/gopsutil/load"
)

var lastPing time.Time
var m sync.Mutex

func Ping(rw http.ResponseWriter, req *http.Request) {
	// Get system load average of the last 5 minutes and compare it against a threashold.

	a, err := load.Avg()
	if err != nil {
		log.Println("Cannot get system load average!", err)
	} else {
		if a.Load5 > config.MaxLoadAvg {
			log.Printf("System load average is too high [%f]\n", a.Load5)
			rw.WriteHeader(http.StatusInsufficientStorage)
			return
		}
	}

	m.Lock()
	defer m.Unlock()

	if time.Now().Sub(lastPing) > 1*time.Minute {
		s, err := services.NewSession(30 * time.Second)
		if err != nil {
			log.Printf("Error creating session [%s]\n", err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, err = services.NewInstance(s, "")

		if err != nil {
			log.Printf("Error creating instance for session [%s]\n", err)
			if cerr := services.CloseSession(s); cerr != nil {
				log.Printf("Error closing session [%s]\n", cerr)
				rw.WriteHeader(http.StatusInternalServerError)
				return
			}
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		lastPing = time.Now()
	}

}
