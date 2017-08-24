package provisioner

type instanceProvisionerFactory struct {
	windows InstanceProvisionerApi
	dind    InstanceProvisionerApi
}

func NewInstanceProvisionerFactory(w InstanceProvisionerApi, d InstanceProvisionerApi) InstanceProvisionerFactoryApi {
	return &instanceProvisionerFactory{windows: w, dind: d}
}

func (p *instanceProvisionerFactory) GetProvisioner(instanceType string) (InstanceProvisionerApi, error) {
	if instanceType == "windows" {
		return p.windows, nil
	} else {
		return p.dind, nil
	}
}
