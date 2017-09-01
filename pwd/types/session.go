package types

import (
	"sync"
	"time"
)

type Session struct {
	Id           string     `json:"id" bson:"id"`
	CreatedAt    time.Time  `json:"created_at"`
	ExpiresAt    time.Time  `json:"expires_at"`
	PwdIpAddress string     `json:"pwd_ip_address"`
	Ready        bool       `json:"ready"`
	Stack        string     `json:"stack"`
	StackName    string     `json:"stack_name"`
	ImageName    string     `json:"image_name"`
	Host         string     `json:"host"`
	rw           sync.Mutex `json:"-"`
}

func (s *Session) Lock() {
	s.rw.Lock()
}

func (s *Session) Unlock() {
	s.rw.Unlock()
}
