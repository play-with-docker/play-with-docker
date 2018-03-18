package main

import (
	"testing"

	"github.com/play-with-docker/play-with-docker/router"
	"github.com/stretchr/testify/assert"
)

func TestDirector(t *testing.T) {
	info, err := director(router.ProtocolHTTP, "ip10-0-0-1-aabb-8080.foo.bar")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.1:8080", info.Dst.String())

	info, err = director(router.ProtocolHTTP, "ip10-0-0-1-aabb.foo.bar")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.1:80", info.Dst.String())

	info, err = director(router.ProtocolHTTPS, "ip10-0-0-1-aabb.foo.bar")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.1:443", info.Dst.String())

	info, err = director(router.ProtocolSSH, "ip10-0-0-1-aabb.foo.bar")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.1:22", info.Dst.String())
	assert.Equal(t, "root", info.SSHUser)

	info, err = director(router.ProtocolDNS, "ip10-0-0-1-aabb.foo.bar")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.1:53", info.Dst.String())

	info, err = director(router.ProtocolHTTP, "ip10-0-0-1-aabb.foo.bar:9090")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.1:9090", info.Dst.String())

	info, err = director(router.ProtocolHTTP, "ip10-0-0-1-aabb-2222.foo.bar:9090")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.1:2222", info.Dst.String())

	info, err = director(router.ProtocolHTTP, "lala.ip10-0-0-1-aabb-2222.foo.bar")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.1:2222", info.Dst.String())

	info, err = director(router.ProtocolHTTP, "lala.ip10-0-0-1-aabb-2222")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.1:2222", info.Dst.String())

	info, err = director(router.ProtocolHTTP, "ip10-0-0-1-aabb-2222")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.1:2222", info.Dst.String())

	info, err = director(router.ProtocolHTTP, "ip10-0-0-1-aabb")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.1:80", info.Dst.String())

	_, err = director(router.ProtocolHTTP, "lala10-0-0-1-aabb.foo.bar")
	assert.NotNil(t, err)
}
