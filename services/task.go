package services

type periodicTask interface {
	Run(i *Instance)
}

var periodicTasks []periodicTask

func init() {
	periodicTasks = append(periodicTasks, &collectStatsTask{}, &checkSwarmStatusTask{}, &broadcastInfoTask{})
}
