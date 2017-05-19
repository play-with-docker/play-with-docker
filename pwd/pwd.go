package pwd

import (
	"sync"
	"time"

	"github.com/play-with-docker/play-with-docker/docker"
)

type Session struct {
	rw           sync.Mutex
	Id           string               `json:"id"`
	Instances    map[string]*Instance `json:"instances"`
	clients      []*Client            `json:"-"`
	CreatedAt    time.Time            `json:"created_at"`
	ExpiresAt    time.Time            `json:"expires_at"`
	scheduled    bool                 `json:"-"`
	ticker       *time.Ticker         `json:"-"`
	PwdIpAddress string               `json:"pwd_ip_address"`
	Ready        bool                 `json:"ready"`
	Stack        string               `json:"stack"`
	closingTimer *time.Timer          `json:"-"`
}

type Instance struct {
}

type Client struct {
}

type pwd struct {
	docker docker.Docker `json:"-"`
}

type PWDApi interface {
	NewSession(duration time.Duration, stack string) (*Session, error)
}

func NewPWD(d docker.Docker) pwd {
	return pwd{docker: d}
}
