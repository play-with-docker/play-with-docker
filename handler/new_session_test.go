package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/franela/play-with-docker/core"
	"github.com/stretchr/testify/assert"
)

func TestNewSession(t *testing.T) {
	mockC := &mockCore{}
	mockR := &mockRecaptcha{}

	conf := NewConfig()
	conf.RootPath = ".."
	h, _ := New(conf, mockC, mockR)
	ts := httptest.NewServer(http.Handler(h))
	defer ts.Close()

	// Status 500 on unknown errors from recaptcha
	mockR.isHuman = func(req *http.Request) (bool, error) {
		return false, fmt.Errorf("unknown error")
	}
	req, e := http.NewRequest("POST", ts.URL, nil)
	assert.Nil(t, e)
	res, err := http.DefaultClient.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusInternalServerError, res.StatusCode)

	// Status 409 when not human
	mockR.isHuman = func(req *http.Request) (bool, error) {
		return false, nil
	}
	req, e = http.NewRequest("POST", ts.URL, nil)
	assert.Nil(t, e)
	res, err = http.DefaultClient.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusConflict, res.StatusCode)

	// Status 500 when cannot create a session
	mockR.isHuman = func(req *http.Request) (bool, error) {
		return true, nil
	}
	mockC.newSession = func() (*core.Session, error) {
		return nil, fmt.Errorf("unknown error")
	}
	req, e = http.NewRequest("POST", ts.URL, nil)
	assert.Nil(t, e)
	res, err = http.DefaultClient.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusInternalServerError, res.StatusCode)

	// Redirect when successfully created a session
	mockR.isHuman = func(req *http.Request) (bool, error) {
		return true, nil
	}
	mockC.newSession = func() (*core.Session, error) {
		return &core.Session{Id: "123456"}, nil
	}
	req, e = http.NewRequest("POST", ts.URL, nil)
	assert.Nil(t, e)

	// Make it so it doesn't follow redirects
	http.DefaultClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	res, err = http.DefaultClient.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusFound, res.StatusCode)
	assert.Equal(t, res.Header.Get("Location"), "/p/123456")
}
