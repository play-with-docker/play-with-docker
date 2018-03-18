package types

import (
	"time"
)

type SessionConfig struct {
	Playground *Playground
	UserId     string
	Duration   time.Duration
	Stack      string
	StackName  string
	ImageName  string
}

type Session struct {
	Id           string    `json:"id" bson:"id"`
	CreatedAt    time.Time `json:"created_at" bson:"created_at"`
	ExpiresAt    time.Time `json:"expires_at" bson:"expires_at"`
	PwdIpAddress string    `json:"pwd_ip_address" bson:"pwd_ip_address"`
	Ready        bool      `json:"ready" bson:"ready"`
	Stack        string    `json:"stack" bson:"stack"`
	StackName    string    `json:"stack_name" bson:"stack_name"`
	ImageName    string    `json:"image_name" bson:"image_name"`
	Host         string    `json:"host" bson:"host"`
	UserId       string    `json:"user_id" bson:"user_id"`
	PlaygroundId string    `json:"playground_id" bson:"playground_id"`
}
