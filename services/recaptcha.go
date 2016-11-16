package services

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func GetGoogleRecaptchaSiteKey() string {
	key := os.Getenv("GOOGLE_RECAPTCHA_SITE_KEY")
	if key == "" {
		// This is a development default. The environment variable should always be set in production.
		key = "6LeY_QsUAAAAAOlpVw4MhoLEr50h-dM80oz6M2AX"
	}
	return key
}
func GetGoogleRecaptchaSiteSecret() string {
	key := os.Getenv("GOOGLE_RECAPTCHA_SITE_SECRET")
	if key == "" {
		// This is a development default. The environment variable should always be set in production.
		key = "6LeY_QsUAAAAAHIALCtm0GKfk-UhtXoyJKarnRV8"
	}

	return key
}

type recaptchaResponse struct {
	Success bool `json:"success"`
}

func IsHuman(req *http.Request) bool {
	req.ParseForm()
	challenge := req.Form.Get("g-recaptcha-response")

	// Of X-Forwarded-For exists, it means we are behind a loadbalancer and we should use the real IP address of the user
	ip := req.Header.Get("X-Forwarded-For")
	if ip == "" {
		// Use the standard remote IP address of the request

		ip = req.RemoteAddr
	}

	parts := strings.Split(ip, ":")

	resp, postErr := http.PostForm("https://www.google.com/recaptcha/api/siteverify", url.Values{"secret": {GetGoogleRecaptchaSiteSecret()}, "response": {challenge}, "remoteip": {parts[0]}})
	if postErr != nil {
		log.Println(postErr)
		// If there is a problem to connect to google, assume the user is a human so we don't block real users because of technical issues
		return true
	}

	var r recaptchaResponse
	json.NewDecoder(resp.Body).Decode(&r)

	return r.Success
}
