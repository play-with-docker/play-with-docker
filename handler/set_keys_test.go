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

	// Status 400 on an empty json in body
	req, e := http.NewRequest("POST", fmt.Sprintf("%s/sessions/no-session/instances/no-instance/keys", ts.URL), nil)
	assert.Nil(t, e)
	res, err := http.DefaultClient.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusBadRequest, res.StatusCode)

	// Status 400 on bad json in body
	body := strings.NewReader(`{}`)
	req, e = http.NewRequest("POST", fmt.Sprintf("%s/sessions/no-session/instances/no-instance/keys", ts.URL), body)
	assert.Nil(t, e)
	res, err = http.DefaultClient.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusBadRequest, res.StatusCode)

	// Status 500 on unknown errors
	body = strings.NewReader(`{"server_cert": "Ymxh", "server_key": "Ymxh"}`)
	mockC.setInstanceCertificate = func(sessionId, instanceName string, cert, key []byte) error {
		return fmt.Errorf("unknown error")
	}
	req, e = http.NewRequest("POST", fmt.Sprintf("%s/sessions/no-session/instances/no-instance/keys", ts.URL), body)
	assert.Nil(t, e)
	res, err = http.DefaultClient.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusInternalServerError, res.StatusCode)

	// Status 404 when session is not found
	body = strings.NewReader(`{"server_cert": "Ymxh", "server_key": "Ymxh"}`)
	mockC.setInstanceCertificate = func(sessionId, instanceName string, cert, key []byte) error {
		return core.NewSessionNotFound(sessionId)
	}
	req, e = http.NewRequest("POST", fmt.Sprintf("%s/sessions/no-session/instances/no-instance/keys", ts.URL), body)
	assert.Nil(t, e)
	res, err = http.DefaultClient.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusNotFound, res.StatusCode)

	// Status 404 when instance is not found
	body = strings.NewReader(`{"server_cert": "Ymxh", "server_key": "Ymxh"}`)
	mockC.setInstanceCertificate = func(sessionId, instanceName string, cert, key []byte) error {
		return core.NewInstanceNotFound(instanceName)
	}
	req, e = http.NewRequest("POST", fmt.Sprintf("%s/sessions/no-session/instances/no-instance/keys", ts.URL), body)
	assert.Nil(t, e)
	res, err = http.DefaultClient.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusNotFound, res.StatusCode)

	// Status 200 when keys are set successfully
	body = strings.NewReader(`{"server_cert": "Zm9v", "server_key": "YmFy"}`)
	var actualCert, actualKey []byte
	mockC.setInstanceCertificate = func(sessionId, instanceName string, cert, key []byte) error {
		actualCert = cert
		actualKey = key
		return nil
	}
	req, e = http.NewRequest("POST", fmt.Sprintf("%s/sessions/session/instances/instance/keys", ts.URL), body)
	assert.Nil(t, e)
	res, err = http.DefaultClient.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, []byte("foo"), actualCert)
	assert.Equal(t, []byte("bar"), actualKey)
}
