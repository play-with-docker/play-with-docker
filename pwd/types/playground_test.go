package types

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestPlayground_Extras_GetInt(t *testing.T) {
	p := Playground{
		Id:     uuid.NewV4().String(),
		Domain: "localhost",
		DefaultDinDInstanceImage: "franel/dind",
		AllowWindowsInstances:    false,
		DefaultSessionDuration:   time.Hour * 4,
		Extras: PlaygroundExtras{
			"intFromInt":    10,
			"intFromFloat":  32.0,
			"intFromString": "15",
		},
	}

	b, err := json.Marshal(p)
	assert.Nil(t, err)

	var p2 Playground
	json.Unmarshal(b, &p2)

	v, found := p2.Extras.GetInt("intFromInt")
	assert.True(t, found)
	assert.Equal(t, 10, v)

	v, found = p2.Extras.GetInt("intFromFloat")
	assert.True(t, found)
	assert.Equal(t, 32, v)

	v, found = p2.Extras.GetInt("intFromString")
	assert.True(t, found)
	assert.Equal(t, 15, v)
}

func TestPlayground_Extras_GetString(t *testing.T) {
	p := Playground{
		Id:     uuid.NewV4().String(),
		Domain: "localhost",
		DefaultDinDInstanceImage: "franel/dind",
		AllowWindowsInstances:    false,
		DefaultSessionDuration:   time.Hour * 4,
		Extras: PlaygroundExtras{
			"stringFromInt":    10,
			"stringFromFloat":  32.3,
			"stringFromString": "15",
			"stringFromBool":   false,
		},
	}

	b, err := json.Marshal(p)
	assert.Nil(t, err)

	var p2 Playground
	json.Unmarshal(b, &p2)

	v, found := p2.Extras.GetString("stringFromInt")
	assert.True(t, found)
	assert.Equal(t, "10", v)

	v, found = p2.Extras.GetString("stringFromFloat")
	assert.True(t, found)
	assert.Equal(t, "32.3", v)

	v, found = p2.Extras.GetString("stringFromString")
	assert.True(t, found)
	assert.Equal(t, "15", v)

	v, found = p2.Extras.GetString("stringFromBool")
	assert.True(t, found)
	assert.Equal(t, "false", v)
}

func TestPlayground_Extras_GetDuration(t *testing.T) {
	p := Playground{
		Id:     uuid.NewV4().String(),
		Domain: "localhost",
		DefaultDinDInstanceImage: "franel/dind",
		AllowWindowsInstances:    false,
		DefaultSessionDuration:   time.Hour * 4,
		Extras: PlaygroundExtras{
			"durationFromInt":      10,
			"durationFromFloat":    32.3,
			"durationFromString":   "4h",
			"durationFromDuration": time.Hour * 3,
		},
	}

	b, err := json.Marshal(p)
	assert.Nil(t, err)

	var p2 Playground
	json.Unmarshal(b, &p2)

	v, found := p2.Extras.GetDuration("durationFromInt")
	assert.True(t, found)
	assert.Equal(t, time.Duration(10), v)

	v, found = p2.Extras.GetDuration("durationFromFloat")
	assert.True(t, found)
	assert.Equal(t, time.Duration(32), v)

	v, found = p2.Extras.GetDuration("durationFromString")
	assert.True(t, found)
	assert.Equal(t, time.Hour*4, v)

	v, found = p2.Extras.GetDuration("durationFromDuration")
	assert.True(t, found)
	assert.Equal(t, time.Hour*3, v)
}
