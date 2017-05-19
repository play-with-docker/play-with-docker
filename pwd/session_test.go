package pwd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewSession_WithoutStack(t *testing.T) {
	createdNetworkId := ""

	mock := &mockDocker{}
	mock.createNetwork = func(id string) error {
		createdNetworkId = id
		return nil
	}

	p := NewPWD(mock)

	before := time.Now()
	s, e := p.NewSession(time.Hour, "", "")

	assert.Nil(t, e)
	assert.NotNil(t, s)

	assert.NotEmpty(t, s.Id)
	assert.WithinDuration(t, s.CreatedAt, before, time.Since(before))
	assert.WithinDuration(t, s.ExpiresAt, before.Add(time.Hour), time.Second)
	assert.Equal(t, s.Id, createdNetworkId)

	assert.NotNil(t, s.closingTimer)
}
