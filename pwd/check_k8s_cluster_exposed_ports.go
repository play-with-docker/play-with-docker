package pwd

import meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type checkK8sClusterExposedPortsTask struct {
}

func (c checkK8sClusterExposedPortsTask) Run(i *Instance) error {
	if i.k8s == nil {
		return nil
	}

	if i.IsManager != nil && *i.IsManager == false {
		return nil
	}

	list, err := i.k8s.CoreV1().Services("").List(meta_v1.ListOptions{})
	if err != nil {
		return err
	}
	exposedPorts := []int32{}

	for _, svc := range list.Items {
		for _, p := range svc.Spec.Ports {
			if p.NodePort > 0 {
				exposedPorts = append(exposedPorts, p.NodePort)
			}
		}
	}

	nodeList, err := i.k8s.CoreV1().Nodes().List(meta_v1.ListOptions{})
	if err != nil {
		return err
	}
	for _, node := range nodeList.Items {
		for _, ins := range i.session.Instances {
			if node.Name == ins.Hostname {
				for _, p := range exposedPorts {
					ins.setUsedPort(uint16(p))
				}
			}
		}
	}
	return nil
}
