package services

func InstanceImages() []string {

	return []string{
		defaultDindImageName,
		"franela/dind:overlay2-dev",
	}

}
