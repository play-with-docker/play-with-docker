package pwd

import meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type checkK8sClusterStatusTask struct {
}

func (c checkK8sClusterStatusTask) Run(i *Instance) error {
	t := true
	f := false

	if i.k8s == nil {
		return nil
	}

	if i.IsManager != nil && *i.IsManager == false {
		return nil
	}

	list, err := i.k8s.CoreV1().Nodes().List(meta_v1.ListOptions{})
	if err != nil {
		return err
	}
	i.IsManager = &t

	for _, node := range list.Items {
		for _, ins := range i.session.Instances {
			if node.Name == ins.Hostname && ins.Name != i.Name {
				ins.IsManager = &f
			}
		}
	}

	return nil
}
