package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/franela/play-with-docker/core"
	"github.com/stretchr/testify/assert"
)

func TestNewInstance(t *testing.T) {
	mockC := &mockCore{}
	mockR := &mockRecaptcha{}

	conf := NewConfig()
	conf.RootPath = ".."
	h, _ := New(conf, mockC, mockR)
	ts := httptest.NewServer(http.Handler(h))
	defer ts.Close()

	// Status 500 on unknown errors
	mockC.getSession = func(sessionId string) (*core.Session, error) {
		return nil, fmt.Errorf("unknown error")
	}
	req, e := http.NewRequest("POST", fmt.Sprintf("%s/sessions/no-session/instances", ts.URL), nil)
	assert.Nil(t, e)
	res, err := http.DefaultClient.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, res.StatusCode, http.StatusInternalServerError)

	// Status 404 when session doesn't exist
	mockC.getSession = func(sessionId string) (*core.Session, error) {
		return nil, core.NewSessionNotFound(sessionId)
	}
	req, e = http.NewRequest("POST", fmt.Sprintf("%s/sessions/no-session/instances", ts.URL), nil)
	assert.Nil(t, e)
	res, err = http.DefaultClient.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, res.StatusCode, http.StatusNotFound)

	// Status 409 when max instances was reached in a session
	mockC.getSession = func(sessionId string) (*core.Session, error) {
		return &core.Session{}, nil
	}
	mockC.newInstance = func(session *core.Session) (*core.Instance, error) {
		return nil, core.NewMaxInstancesInSessionReached()
	}
	req, e = http.NewRequest("POST", fmt.Sprintf("%s/sessions/valid-session/instances", ts.URL), nil)
	assert.Nil(t, e)
	res, err = http.DefaultClient.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, res.StatusCode, http.StatusConflict)

	// Status 200 and valid instance json
	mockC.getSession = func(sessionId string) (*core.Session, error) {
		return &core.Session{}, nil
	}
	expectedInstance := &core.Instance{}
	mockC.newInstance = func(session *core.Session) (*core.Instance, error) {
		return expectedInstance, nil
	}
	req, e = http.NewRequest("POST", fmt.Sprintf("%s/sessions/valid-session/instances", ts.URL), nil)
	assert.Nil(t, e)
	res, err = http.DefaultClient.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	var actualInstance core.Instance
	jsonErr := json.NewDecoder(res.Body).Decode(&actualInstance)

	assert.Nil(t, jsonErr)
	assert.Equal(t, *expectedInstance, actualInstance)
}
