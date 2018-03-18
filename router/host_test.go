package router

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeHostInfo(t *testing.T) {
	host := EncodeHost("aaabbbcccddd", "10.0.0.1", HostOpts{})
	assert.Equal(t, "ip10-0-0-1-aaabbbcccddd", host)

	opts := HostOpts{EncodedPort: 8080}
	host = EncodeHost("aaabbbcccddd", "10.0.0.1", opts)
	assert.Equal(t, "ip10-0-0-1-aaabbbcccddd-8080", host)

	opts = HostOpts{TLD: "foo.bar"}
	host = EncodeHost("aaabbbcccddd", "10.0.0.1", opts)
	assert.Equal(t, "ip10-0-0-1-aaabbbcccddd.foo.bar", host)

	opts = HostOpts{TLD: "foo.bar", EncodedPort: 8080, Port: 443}
	host = EncodeHost("aaabbbcccddd", "10.0.0.1", opts)
	assert.Equal(t, "ip10-0-0-1-aaabbbcccddd-8080.foo.bar:443", host)
}

func TestDecodeHostInfo(t *testing.T) {
	info, err := DecodeHost("ip10-0-0-1-aaabbbcccddd")
	assert.Nil(t, err)
	assert.Equal(t, HostInfo{InstanceIP: "10.0.0.1", SessionId: "aaabbbcccddd"}, info)

	info, err = DecodeHost("ip10-0-0-1-aaabbbcccddd-8080")
	assert.Nil(t, err)
	assert.Equal(t, HostInfo{InstanceIP: "10.0.0.1", SessionId: "aaabbbcccddd", EncodedPort: 8080}, info)

	info, err = DecodeHost("ip10-0-0-1-aaabbbcccddd-8080.foo.bar")
	assert.Nil(t, err)
	assert.Equal(t, HostInfo{InstanceIP: "10.0.0.1", SessionId: "aaabbbcccddd", EncodedPort: 8080, TLD: "foo.bar"}, info)

	info, err = DecodeHost("ip10-0-0-1-aaabbbcccddd-8080.foo.bar:443")
	assert.Nil(t, err)
	assert.Equal(t, HostInfo{InstanceIP: "10.0.0.1", SessionId: "aaabbbcccddd", EncodedPort: 8080, TLD: "foo.bar", Port: 443}, info)

	_, err = DecodeHost("ip10-0-0-1")
	assert.NotNil(t, err)
}
