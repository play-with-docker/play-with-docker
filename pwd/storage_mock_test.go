package pwd

type mockStorage struct {
	save func() error
	load func() error
}

func (m *mockStorage) Save() error {
	if m.save != nil {
		return m.save()
	}
	return nil
}
func (m *mockStorage) Load() error {
	if m.load != nil {
		return m.load()
	}
	return nil
}
