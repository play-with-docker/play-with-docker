package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/franela/play-with-docker/core"
	"github.com/stretchr/testify/assert"
)

func TestGetSession(t *testing.T) {
	mockC := &mockCore{}
	mockR := &mockRecaptcha{}

	conf := NewConfig()
	conf.RootPath = ".."
	h, _ := New(conf, mockC, mockR)
	ts := httptest.NewServer(http.Handler(h))
	defer ts.Close()

	// Test 500 error
	mockC.getSession = func(sessionId string) (*core.Session, error) {
		return nil, fmt.Errorf("unknown error")
	}
	req, e := http.NewRequest("GET", fmt.Sprintf("%s/sessions/no-session", ts.URL), nil)
	assert.Nil(t, e)
	res, err := http.DefaultClient.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, res.StatusCode, http.StatusInternalServerError)

	// Test 404 error when session not found
	mockC.getSession = func(sessionId string) (*core.Session, error) {
		return nil, core.NewSessionNotFound(sessionId)
	}
	req, e = http.NewRequest("GET", fmt.Sprintf("%s/sessions/no-session", ts.URL), nil)
	assert.Nil(t, e)
	res, err = http.DefaultClient.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, res.StatusCode, http.StatusNotFound)

	// Test 200 status and session json structure
	expected := &core.Session{
		Id:        "123",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	mockC.getSession = func(sessionId string) (*core.Session, error) {
		return expected, nil
	}
	req, e = http.NewRequest("GET", fmt.Sprintf("%s/sessions/valid-session", ts.URL), nil)
	assert.Nil(t, e)
	res, err = http.DefaultClient.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)

	var actual core.Session
	jsonErr := json.NewDecoder(res.Body).Decode(&actual)

	assert.Nil(t, jsonErr)
	assert.Equal(t, *expected, actual)
}
