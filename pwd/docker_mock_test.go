package pwd

type mockDocker struct {
	createNetwork func(string) error
}

func (m *mockDocker) CreateNetwork(id string) error {
	return m.createNetwork(id)
}
