package services

type periodicTask interface {
	Run(i *Instance) error
}

var periodicTasks []periodicTask

func init() {
	periodicTasks = append(periodicTasks, &collectStatsTask{}, &checkSwarmStatusTask{}, &checkUsedPortsTask{}, &checkSwarmUsedPortsTask{})
}
