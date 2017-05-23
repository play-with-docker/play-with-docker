package pwd

type mockTasks struct {
	schedule   func(s *Session)
	unschedule func(s *Session)
}

func (m *mockTasks) Schedule(s *Session) {
	if m.schedule != nil {
		m.schedule(s)
	}
}
func (m *mockTasks) Unschedule(s *Session) {
	if m.unschedule != nil {
		m.unschedule(s)
	}
}
