package main

import (
	"log"
	"os"

	"github.com/play-with-docker/play-with-docker/config"
	"github.com/play-with-docker/play-with-docker/docker"
	"github.com/play-with-docker/play-with-docker/event"
	"github.com/play-with-docker/play-with-docker/handlers"
	"github.com/play-with-docker/play-with-docker/provisioner"
	"github.com/play-with-docker/play-with-docker/pwd"
	"github.com/play-with-docker/play-with-docker/scheduler"
	"github.com/play-with-docker/play-with-docker/scheduler/task"
	"github.com/play-with-docker/play-with-docker/storage"
)

func main() {
	config.ParseFlags()

	e := initEvent()
	s := initStorage()
	f := initFactory(s)

	ipf := provisioner.NewInstanceProvisionerFactory(provisioner.NewWindows(f, s), provisioner.NewDinD(f, s))
	sp := provisioner.NewOverlaySessionProvisioner(f)

	core := pwd.NewPWD(f, e, s, sp, ipf)

	sch, err := scheduler.NewScheduler(s, e, core)
	if err != nil {
		log.Fatal("Error initializing the scheduler: ", err)
	}

	sch.AddTask(task.NewCheckPorts(e, f))
	sch.AddTask(task.NewCheckSwarmPorts(e, f))
	sch.AddTask(task.NewCheckSwarmStatus(e, f))
	sch.AddTask(task.NewCollectStats(e, f))

	sch.Start()

	handlers.Bootstrap(core, e)
	handlers.Register()
}

func initStorage() storage.StorageApi {
	s, err := storage.NewFileStorage(config.SessionsFile)
	if err != nil && !os.IsNotExist(err) {
		log.Fatal("Error initializing StorageAPI: ", err)
	}
	return s
}

func initEvent() event.EventApi {
	return event.NewLocalBroker()
}

func initFactory(s storage.StorageApi) docker.FactoryApi {
	return docker.NewLocalCachedFactory(s)
}
