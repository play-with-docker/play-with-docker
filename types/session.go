package types

import (
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/franela/play-with-docker/cookoo"
)

type Session struct {
	sync.Mutex
	Id        string               `json:"id"`
	Instances map[string]*Instance `json:"instances"`
}

type Instance struct {
	Name   string                  `json:"name"`
	IP     string                  `json:"ip"`
	Stdout *cookoo.MultiWriter     `json:"-"`
	Conn   *types.HijackedResponse `json:"-"`
}
