package pwd

type mockStorage struct {
}

func (m *mockStorage) Save() error {
	return nil
}
func (m *mockStorage) Load() error {
	return nil
}
