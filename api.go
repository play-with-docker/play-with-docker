package main

import (
	"log"
	"os"
	"time"

	"github.com/play-with-docker/play-with-docker/config"
	"github.com/play-with-docker/play-with-docker/docker"
	"github.com/play-with-docker/play-with-docker/event"
	"github.com/play-with-docker/play-with-docker/handlers"
	"github.com/play-with-docker/play-with-docker/id"
	"github.com/play-with-docker/play-with-docker/k8s"
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
	df := initDockerFactory(s)
	kf := initK8sFactory(s)

	ipf := provisioner.NewInstanceProvisionerFactory(provisioner.NewWindowsASG(df, s), provisioner.NewDinD(id.XIDGenerator{}, df, s))
	sp := provisioner.NewOverlaySessionProvisioner(df)

	core := pwd.NewPWD(df, e, s, sp, ipf)

	tasks := []scheduler.Task{
		task.NewCheckPorts(e, df),
		task.NewCheckSwarmPorts(e, df),
		task.NewCheckSwarmStatus(e, df),
		task.NewCollectStats(e, df, s),
		task.NewCheckK8sClusterStatus(e, kf),
		task.NewCheckK8sClusterExposedPorts(e, kf),
	}
	sch, err := scheduler.NewScheduler(tasks, s, e, core)
	if err != nil {
		log.Fatal("Error initializing the scheduler: ", err)
	}

	sch.Start()

	d, err := time.ParseDuration(config.DefaultSessionDuration)
	if err != nil {
		log.Fatalf("Cannot parse duration %s. Got: %v", config.DefaultSessionDuration, err)
	}

	playground := types.Playground{Domain: config.PlaygroundDomain, DefaultDinDInstanceImage: config.DefaultDinDImage, AllowWindowsInstances: config.NoWindows, DefaultSessionDuration: d, AvailableDinDInstanceImages: []string{config.DefaultDinDImage}, Tasks: []string{".*"}}
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

func initDockerFactory(s storage.StorageApi) docker.FactoryApi {
	return docker.NewLocalCachedFactory(s)
}

func initK8sFactory(s storage.StorageApi) k8s.FactoryApi {
	return k8s.NewLocalCachedFactory(s)
}
