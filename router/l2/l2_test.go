package main

import (
	"testing"

	"github.com/play-with-docker/play-with-docker/router"
	"github.com/stretchr/testify/assert"
)

func TestDirector(t *testing.T) {
	addr, err := director(router.ProtocolHTTP, "ip10-0-0-1-aabb-8080.foo.bar")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.1:8080", addr.String())

	addr, err = director(router.ProtocolHTTP, "ip10-0-0-1-aabb.foo.bar")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.1:80", addr.String())

	addr, err = director(router.ProtocolHTTPS, "ip10-0-0-1-aabb.foo.bar")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.1:443", addr.String())

	addr, err = director(router.ProtocolSSH, "ip10-0-0-1-aabb.foo.bar")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.1:22", addr.String())

	addr, err = director(router.ProtocolDNS, "ip10-0-0-1-aabb.foo.bar")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.1:53", addr.String())

	addr, err = director(router.ProtocolHTTP, "ip10-0-0-1-aabb.foo.bar:9090")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.1:9090", addr.String())

	addr, err = director(router.ProtocolHTTP, "ip10-0-0-1-aabb-2222.foo.bar:9090")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.1:2222", addr.String())

	addr, err = director(router.ProtocolHTTP, "lala.ip10-0-0-1-aabb-2222.foo.bar")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.1:2222", addr.String())

	addr, err = director(router.ProtocolHTTP, "lala.ip10-0-0-1-aabb-2222")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.1:2222", addr.String())

	addr, err = director(router.ProtocolHTTP, "ip10-0-0-1-aabb-2222")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.1:2222", addr.String())

	addr, err = director(router.ProtocolHTTP, "ip10-0-0-1-aabb")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.1:80", addr.String())

	_, err = director(router.ProtocolHTTP, "lala10-0-0-1-aabb.foo.bar")
	assert.NotNil(t, err)
}
