package services

type broadcastInfoTask struct {
}

func (c *broadcastInfoTask) Run(i *Instance) {
	wsServer.BroadcastTo(i.session.Id, "instance stats", i.Name, i.Mem, i.Cpu, i.IsManager)
}
