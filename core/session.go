package core

import "time"

type Session struct {
	Id        string               `json:"id"`
	Instances map[string]*Instance `json:"instances"`
	CreatedAt time.Time            `json:"created_at"`
	ExpiresAt time.Time            `json:"expires_at"`
}
