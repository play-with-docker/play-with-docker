package types

import (
	"sync"
	"time"
)

type Session struct {
	Id           string               `json:"id"`
	Instances    map[string]*Instance `json:"instances"`
	CreatedAt    time.Time            `json:"created_at"`
	ExpiresAt    time.Time            `json:"expires_at"`
	PwdIpAddress string               `json:"pwd_ip_address"`
	Ready        bool                 `json:"ready"`
	Stack        string               `json:"stack"`
	StackName    string               `json:"stack_name"`
	ImageName    string               `json:"image_name"`
	Host         string               `json:"host"`
	Clients      []*Client            `json:"-"`
	closingTimer *time.Timer          `json:"-"`
	scheduled    bool                 `json:"-"`
	ticker       *time.Ticker         `json:"-"`
	rw           sync.Mutex           `json:"-"`
	prepared     bool                 `json:"-"`
}

func (s *Session) Lock() {
	s.rw.Lock()
}

func (s *Session) Unlock() {
	s.rw.Unlock()
}

func (s *Session) StopTicker() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
}
func (s *Session) SetTicker(t *time.Ticker) {
	s.ticker = t
}
func (s *Session) SetClosingTimer(t *time.Timer) {
	s.closingTimer = t
}
func (s *Session) ClosingTimer() *time.Timer {
	return s.closingTimer
}

func (s *Session) IsPrepared() bool {
	return s.prepared
}

func (s *Session) SetPrepared() {
	s.prepared = true
}
