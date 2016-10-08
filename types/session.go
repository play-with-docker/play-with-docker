package types

import (
	"github.com/docker/docker/api/types"
	"github.com/franela/play-with-docker/cookoo"
)

type Session struct {
	Id        string               `json:"id"`
	Instances map[string]*Instance `json:"instances"`
}

type Instance struct {
	Name   string                  `json:"name"`
	IP     string                  `json:"ip"`
	Stdout *cookoo.MultiWriter     `json:"-"`
	Conn   *types.HijackedResponse `json:"-"`
}
