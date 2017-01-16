package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/franela/play-with-docker/core"
	"github.com/stretchr/testify/assert"
)

func TestSetKeys(t *testing.T) {
	mockC := &mockCore{}
	mockR := &mockRecaptcha{}

	conf := NewConfig()
	conf.RootPath = ".."
	h, _ := New(conf, mockC, mockR)
	ts := httptest.NewServer(http.Handler(h))
	defer ts.Close()

	// Status 400 on bad json in body
	req, e := http.NewRequest("POST", fmt.Sprintf("%s/sessions/no-session/instances/no-instance/keys", ts.URL), nil)
	assert.Nil(t, e)
	res, err := http.DefaultClient.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusBadRequest, res.StatusCode)

	body := strings.NewReader(`{}`)
	// Status 400 on bad json in body
	req, e = http.NewRequest("POST", fmt.Sprintf("%s/sessions/no-session/instances/no-instance/keys", ts.URL), body)
	assert.Nil(t, e)
	res, err = http.DefaultClient.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusBadRequest, res.StatusCode)

	body = strings.NewReader(`{"server_cert": "Ymxh", "server_key": "Ymxh"}`)
	// Status 500 on unknown errors
	mockC.getSession = func(sessionId string) (*core.Session, error) {
		return nil, fmt.Errorf("unknown error")
	}
	req, e = http.NewRequest("POST", fmt.Sprintf("%s/sessions/no-session/instances/no-instance/keys", ts.URL), body)
	assert.Nil(t, e)
	res, err = http.DefaultClient.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusInternalServerError, res.StatusCode)

	// Status 500 on unknown errors
	mockC.getSession = func(sessionId string) (*core.Session, error) {
		return &core.Session{}, nil
	}
	mockC.getInstance = func(session *core.Session, instanceName string) (*core.Instance, error) {
		return nil, fmt.Errorf("unknown error")
	}
	req, e = http.NewRequest("POST", fmt.Sprintf("%s/sessions/no-session/instances/no-instance/keys", ts.URL), body)
	assert.Nil(t, e)
	res, err = http.DefaultClient.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
}
