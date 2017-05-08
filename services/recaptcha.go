package services

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/play-with-docker/play-with-docker/config"
	"github.com/twinj/uuid"
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

var s = securecookie.New([]byte(config.HashKey), nil).MaxAge(int((1 * time.Hour).Seconds()))

func IsHuman(req *http.Request, rw http.ResponseWriter) bool {
	if os.Getenv("GOOGLE_RECAPTCHA_DISABLED") != "" {
		return true
	}

	if cookie, _ := req.Cookie("session_id"); cookie != nil {
		var value string
		if err := s.Decode("session_id", cookie.Value, &value); err != nil {
			fmt.Println(err)
			return false
		}
		return true
	}

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

	if !r.Success {
		return false
	}

	encoded, _ := s.Encode("session_id", uuid.NewV4().String())
	http.SetCookie(rw, &http.Cookie{
		Name:    "session_id",
		Value:   encoded,
		Expires: time.Now().Add(1 * time.Hour),
	})

	return true
}
