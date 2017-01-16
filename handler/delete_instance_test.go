package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/franela/play-with-docker/core"
	"github.com/stretchr/testify/assert"
)

func TestDeleteInstance(t *testing.T) {
	mockC := &mockCore{}
	mockR := &mockRecaptcha{}

	conf := NewConfig()
	conf.RootPath = ".."
	h, _ := New(conf, mockC, mockR)
	ts := httptest.NewServer(http.Handler(h))
	defer ts.Close()

	// Status 500 when on unknown errors
	mockC.deleteInstance = func(sessionId, instanceName string) error {
		return fmt.Errorf("unknown error")
	}
	req, e := http.NewRequest("DELETE", fmt.Sprintf("%s/sessions/no-session/instances/no-instance", ts.URL), nil)
	assert.Nil(t, e)
	res, err := http.DefaultClient.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, res.StatusCode, 500)

	// Status 404 when session doesn't exit
	mockC.deleteInstance = func(sessionId, instanceName string) error {
		return core.NewSessionNotFound(sessionId)
	}
	req, e = http.NewRequest("DELETE", fmt.Sprintf("%s/sessions/no-session/instances/no-instance", ts.URL), nil)
	assert.Nil(t, e)
	res, err = http.DefaultClient.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, res.StatusCode, 404)

	// Status 404 when instance doesn't exit
	mockC.deleteInstance = func(sessionId, instanceName string) error {
		return core.NewInstanceNotFound(sessionId)
	}
	req, e = http.NewRequest("DELETE", fmt.Sprintf("%s/sessions/no-session/instances/no-instance", ts.URL), nil)
	assert.Nil(t, e)
	res, err = http.DefaultClient.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, res.StatusCode, 404)

	// Status 200 when everything is OK
	mockC.deleteInstance = func(sessionId, instanceName string) error {
		return nil
	}
	req, e = http.NewRequest("DELETE", fmt.Sprintf("%s/sessions/a-session/instances/an-instance", ts.URL), nil)
	assert.Nil(t, e)
	res, err = http.DefaultClient.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, res.StatusCode, 200)
}
