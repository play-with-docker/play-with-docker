package handlers

import (
	"log"
	"os"

	"github.com/googollee/go-socket.io"
	"github.com/play-with-docker/play-with-docker/config"
	"github.com/play-with-docker/play-with-docker/docker"
	"github.com/play-with-docker/play-with-docker/event"
	"github.com/play-with-docker/play-with-docker/pwd"
	"github.com/play-with-docker/play-with-docker/scheduler"
	"github.com/play-with-docker/play-with-docker/scheduler/task"
	"github.com/play-with-docker/play-with-docker/storage"
)

var core pwd.PWDApi
var e event.EventApi
var ws *socketio.Server

func Bootstrap() {
	s, err := storage.NewFileStorage(config.SessionsFile)
	e = event.NewLocalBroker()

	f := docker.NewLocalCachedFactory(s)

	if err != nil && !os.IsNotExist(err) {
		log.Fatal("Error initializing StorageAPI: ", err)
	}
	core = pwd.NewPWD(f, e, s)

	sch, err := scheduler.NewScheduler(s, e, core)
	if err != nil {
		log.Fatal("Error initializing the scheduler: ", err)
	}

	sch.AddTask(task.NewCheckPorts(e, f))
	sch.AddTask(task.NewCheckSwarmPorts(e, f))
	sch.AddTask(task.NewCheckSwarmStatus(e, f))
	sch.AddTask(task.NewCollectStats(e, f))

	sch.Start()
}

func RegisterEvents(s *socketio.Server) {
	ws = s
	e.OnAny(broadcastEvent)
}

func broadcastEvent(eventType event.EventType, sessionId string, args ...interface{}) {
	ws.BroadcastTo(sessionId, eventType.String(), args...)
}
