package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestReverseProxy(t *testing.T) {
	mockC := &mockCore{}
	mockR := &mockRecaptcha{}

	conf := NewConfig()
	conf.RootPath = ".."
	h, _ := New(conf, mockC, mockR)
	ts := httptest.NewServer(http.Handler(h))
	defer ts.Close()

	req, e := http.NewRequest("GET", fmt.Sprintf("%s/foo", ts.URL), nil)
	assert.Nil(t, e)

	req.Host = "ip192_168_1_1-5555.play-with-docker.com"
	_, err := http.DefaultClient.Do(req)
	assert.Nil(t, err)
	assert.NotNil(t, mockC.proxiedRequest)
	v := mux.Vars(mockC.proxiedRequest)
	assert.Equal(t, "ip192_168_1_1", v["node"])
	assert.Equal(t, "5555", v["port"])

	req, e = http.NewRequest("GET", fmt.Sprintf("%s/foo", ts.URL), nil)
	assert.Nil(t, e)

	req.Host = "ip192_168_1_1.play-with-docker.com"
	_, err = http.DefaultClient.Do(req)
	assert.Nil(t, err)
	assert.NotNil(t, mockC.proxiedRequest)
	v = mux.Vars(mockC.proxiedRequest)
	assert.Equal(t, "ip192_168_1_1", v["node"])
	assert.Equal(t, "", v["port"])
}

func TestReverseProxySSL(t *testing.T) {
	mockC := &mockCore{}

	h, _ := NewSSL(mockC)
	ts := httptest.NewServer(http.Handler(h))
	defer ts.Close()

	req, e := http.NewRequest("GET", fmt.Sprintf("%s/foo", ts.URL), nil)
	assert.Nil(t, e)

	req.Host = "ip192_168_1_1-2375.play-with-docker.com"
	_, err := http.DefaultClient.Do(req)
	assert.Nil(t, err)
	assert.NotNil(t, mockC.proxiedRequest)
	v := mux.Vars(mockC.proxiedRequest)
	assert.Equal(t, "ip192_168_1_1", v["node"])

	req, e = http.NewRequest("GET", fmt.Sprintf("%s/foo", ts.URL), nil)
	assert.Nil(t, e)

	req.Host = "ip192_168_1_1.play-with-docker.com"
	res, err := http.DefaultClient.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusNotFound, res.StatusCode)
}
