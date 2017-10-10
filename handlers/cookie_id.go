package handlers

import (
	"net/http"

	"github.com/play-with-docker/play-with-docker/config"
)

type CookieID struct {
	Id         string `json:"id"`
	UserName   string `json:"user_name"`
	UserAvatar string `json:"user_avatar"`
}

func (c *CookieID) SetCookie(rw http.ResponseWriter) error {
	if encoded, err := config.SecureCookie.Encode("id", c); err == nil {
		cookie := &http.Cookie{
			Name:     "id",
			Value:    encoded,
			Path:     "/",
			Secure:   config.UseLetsEncrypt,
			HttpOnly: true,
		}
		http.SetCookie(rw, cookie)
	} else {
		return err
	}
	return nil
}
func ReadCookie(r *http.Request) (*CookieID, error) {
	if cookie, err := r.Cookie("id"); err == nil {
		value := &CookieID{}
		if err = config.SecureCookie.Decode("id", cookie.Value, &value); err == nil {
			return value, nil
		} else {
			return nil, err
		}
	} else {
		return nil, err
	}
}
