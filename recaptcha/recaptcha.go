package recaptcha

import "net/http"

type Recaptcha interface {
	IsHuman(req *http.Request) (bool, error)
}

func New() Recaptcha {
	return nil
}
