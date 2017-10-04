package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"golang.org/x/oauth2"

	"github.com/google/go-github/github"
	"github.com/gorilla/mux"
	fb "github.com/huandu/facebook"
	"github.com/play-with-docker/play-with-docker/config"
	"github.com/play-with-docker/play-with-docker/pwd/types"
)

func ListProviders(rw http.ResponseWriter, req *http.Request) {
	providers := []string{}
	for name, _ := range config.Providers {
		providers = append(providers, name)
	}
	json.NewEncoder(rw).Encode(providers)
}

func Login(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	providerName := vars["provider"]

	provider, found := config.Providers[providerName]
	if !found {
		log.Printf("Could not find provider %s\n", providerName)
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	loginRequest, err := core.UserNewLoginRequest(providerName)
	if err != nil {
		log.Printf("Could not start a new user login request for provider %s. Got: %v\n", providerName, err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	scheme := "http"
	if req.URL.Scheme != "" {
		scheme = req.URL.Scheme
	}
	host := "localhost"
	if req.URL.Host != "" {
		host = req.URL.Host
	}
	provider.RedirectURL = fmt.Sprintf("%s://%s/oauth/providers/%s/callback", scheme, host, providerName)
	url := provider.AuthCodeURL(loginRequest.Id)

	http.Redirect(rw, req, url, http.StatusFound)
}

func LoginCallback(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	providerName := vars["provider"]

	provider, found := config.Providers[providerName]
	if !found {
		log.Printf("Could not find provider %s\n", providerName)
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	query := req.URL.Query()

	code := query.Get("code")
	loginRequestId := query.Get("state")

	loginRequest, err := core.UserGetLoginRequest(loginRequestId)
	if err != nil {
		log.Printf("Could not get login request %s for provider %s. Got: %v\n", loginRequestId, providerName, err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	ctx := req.Context()
	tok, err := provider.Exchange(ctx, code)
	if err != nil {
		log.Printf("Could not exchage code for access token for provider %s. Got: %v\n", providerName, err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	user := &types.User{Provider: providerName}
	if providerName == "github" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: tok.AccessToken},
		)
		tc := oauth2.NewClient(ctx, ts)
		client := github.NewClient(tc)
		u, _, err := client.Users.Get(ctx, "")
		if err != nil {
			log.Printf("Could not get user from github. Got: %v\n", err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		user.ProviderUserId = strconv.Itoa(u.GetID())
		user.Name = u.GetName()
		user.Avatar = u.GetAvatarURL()
		user.Email = u.GetEmail()
	} else if providerName == "facebook" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: tok.AccessToken},
		)
		tc := oauth2.NewClient(ctx, ts)
		session := &fb.Session{
			Version:    "v2.10",
			HttpClient: tc,
		}
		p := fb.Params{}
		p["fields"] = "email,name,picture"
		res, err := session.Get("/me", p)
		if err != nil {
			log.Printf("Could not get user from facebook. Got: %v\n", err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		user.ProviderUserId = res.Get("id").(string)
		user.Name = res.Get("name").(string)
		user.Avatar = res.Get("picture.data.url").(string)
		user.Email = res.Get("email").(string)
	}

	user, err = core.UserLogin(loginRequest, user)
	if err != nil {
		log.Printf("Could not login user. Got: %v\n", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	cookieData := CookieID{Id: user.Id, UserName: user.Name, UserAvatar: user.Avatar}

	if err := cookieData.SetCookie(rw); err != nil {
		log.Printf("Could not encode cookie. Got: %v\n", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.Redirect(rw, req, "/", http.StatusFound)
}
