package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDirector(t *testing.T) {
	addr, err := director("ip10-0-0-1-aabb-8080.foo.bar")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.1:8080", addr.String())

	addr, err = director("ip10-0-0-1-aabb.foo.bar")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.1:80", addr.String())

	addr, err = director("ip10-0-0-1-aabb.foo.bar:9090")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.1:9090", addr.String())

	addr, err = director("ip10-0-0-1-aabb-2222.foo.bar:9090")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.1:2222", addr.String())

	addr, err = director("lala.ip10-0-0-1-aabb-2222.foo.bar")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.1:2222", addr.String())

	addr, err = director("lala.ip10-0-0-1-aabb-2222")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.1:2222", addr.String())

	addr, err = director("ip10-0-0-1-aabb-2222")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.1:2222", addr.String())

	addr, err = director("ip10-0-0-1-aabb")
	assert.Nil(t, err)
	assert.Equal(t, "10.0.0.1:80", addr.String())

	_, err = director("lala10-0-0-1-aabb.foo.bar")
	assert.NotNil(t, err)
}
