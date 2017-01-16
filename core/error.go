package core

import (
	"fmt"
	"strings"
)

func NewSessionNotFound(sessionId string) error {
	return fmt.Errorf("Session not found [%s]", sessionId)
}

func SessionNotFound(e error) bool {
	return strings.HasPrefix(e.Error(), "Session not found")
}

func NewInstanceNotFound(instanceName string) error {
	return fmt.Errorf("Instance not found [%s]", instanceName)
}
func InstanceNotFound(e error) bool {
	return strings.HasPrefix(e.Error(), "Instance not found")
}
func NewMaxInstancesInSessionReached() error {
	return fmt.Errorf("Max instances reached on this session")
}
func MaxInstancesInSessionReached(e error) bool {
	return strings.HasPrefix(e.Error(), "Max instances reached on this session")
}
