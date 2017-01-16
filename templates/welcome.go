package templates

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/franela/play-with-docker/services"
)

func GetWelcomeTemplate(rootPath string) ([]byte, error) {
	welcomeTemplate, tplErr := template.New("welcome").ParseFiles(fmt.Sprintf("%s/www/welcome.html", rootPath))
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
