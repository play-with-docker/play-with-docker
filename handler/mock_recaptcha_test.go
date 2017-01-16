package handler

import "net/http"

type mockRecaptcha struct {
	isHuman func(req *http.Request) (bool, error)
}

func (m *mockRecaptcha) IsHuman(req *http.Request) (bool, error) {
	return m.isHuman(req)
}
