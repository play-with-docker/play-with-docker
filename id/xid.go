package id

import "github.com/rs/xid"

type XIDGenerator struct {
}

func (x XIDGenerator) NewId() string {
	return xid.New().String()
}
