package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPing(t *testing.T) {
	mockC := &mockCore{}
	mockR := &mockRecaptcha{}

	conf := NewConfig()
	conf.RootPath = ".."
	h, _ := New(conf, mockC, mockR)
	ts := httptest.NewServer(http.Handler(h))
	defer ts.Close()

	res, err := http.Get(ts.URL)
	assert.Nil(t, err)
	assert.Equal(t, res.StatusCode, http.StatusOK)
}
