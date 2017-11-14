package main

import (
	"log"
	"os"

	"github.com/play-with-docker/play-with-docker/config"
	"github.com/play-with-docker/play-with-docker/docker"
	"github.com/play-with-docker/play-with-docker/event"
	"github.com/play-with-docker/play-with-docker/handlers"
	"github.com/play-with-docker/play-with-docker/id"
	"github.com/play-with-docker/play-with-docker/provisioner"
	"github.com/play-with-docker/play-with-docker/pwd"
	"github.com/play-with-docker/play-with-docker/pwd/types"
	"github.com/play-with-docker/play-with-docker/scheduler"
	"github.com/play-with-docker/play-with-docker/scheduler/task"
	"github.com/play-with-docker/play-with-docker/storage"
)

func main() {
	config.ParseFlags()

	e := initEvent()
	s := initStorage()
	f := initFactory(s)

	ipf := provisioner.NewInstanceProvisionerFactory(provisioner.NewWindowsASG(f, s), provisioner.NewDinD(id.XIDGenerator{}, f, s))
	sp := provisioner.NewOverlaySessionProvisioner(f)

	core := pwd.NewPWD(f, e, s, sp, ipf)

	tasks := []scheduler.Task{
		task.NewCheckPorts(e, f),
		task.NewCheckSwarmPorts(e, f),
		task.NewCheckSwarmStatus(e, f),
		task.NewCollectStats(e, f, s),
	}
	sch, err := scheduler.NewScheduler(tasks, s, e, core)
	if err != nil {
		log.Fatal("Error initializing the scheduler: ", err)
	}

	sch.Start()

	playground := types.Playground{Domain: config.PlaygroundDomain, DefaultDinDInstanceImage: config.GetDindImageName(), AllowWindowsInstances: config.NoWindows, DefaultSessionDuration: config.GetDuration("")}
	if _, err := core.PlaygroundNew(playground); err != nil {
		log.Fatalf("Cannot create default playground. Got: %v", err)
	}

	handlers.Bootstrap(core, e)
	handlers.Register(nil)
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
