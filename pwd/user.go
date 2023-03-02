package pwd

import (
	"errors"

	"github.com/thebsdbox/play-with-docker/pwd/types"
	"github.com/thebsdbox/play-with-docker/storage"
)

var userBannedError = errors.New("User is banned")

func (p *pwd) UserNewLoginRequest(providerName string) (*types.LoginRequest, error) {
	req := &types.LoginRequest{Id: p.generator.NewId(), Provider: providerName}
	if err := p.storage.LoginRequestPut(req); err != nil {
		return nil, err
	}
	return req, nil
}

func (p *pwd) UserGetLoginRequest(id string) (*types.LoginRequest, error) {
	if req, err := p.storage.LoginRequestGet(id); err != nil {
		return nil, err
	} else {
		return req, nil
	}
}

func (p *pwd) UserLogin(loginRequest *types.LoginRequest, user *types.User) (*types.User, error) {
	if err := p.storage.LoginRequestDelete(loginRequest.Id); err != nil {
		return nil, err
	}
	u, err := p.storage.UserFindByProvider(user.Provider, user.ProviderUserId)

	if err != nil {
		if storage.NotFound(err) {
			user.Id = p.generator.NewId()
			if err := p.storage.UserPut(user); err != nil {
				return nil, err
			}
			return user, nil
		}
		return nil, err
	}
	return u, nil
}
func (p *pwd) UserGet(id string) (*types.User, error) {
	var user *types.User
	var err error
	if user, err = p.storage.UserGet(id); err != nil {
		return nil, err
	} else if user.IsBanned {
		return user, userBannedError
	}
	return user, nil
}
