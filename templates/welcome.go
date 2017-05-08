package templates

import (
	"bytes"
	"html/template"

	"github.com/play-with-docker/play-with-docker/services"
)

func GetWelcomeTemplate() ([]byte, error) {
	welcomeTemplate, tplErr := template.New("welcome").ParseFiles("www/welcome.html")
	if tplErr != nil {
		return nil, tplErr
	}
	var b bytes.Buffer
	tplExecuteErr := welcomeTemplate.ExecuteTemplate(&b, "GOOGLE_RECAPTCHA_SITE_KEY", services.GetGoogleRecaptchaSiteKey())
	if tplExecuteErr != nil {
		return nil, tplExecuteErr
	}
	return b.Bytes(), nil
}
