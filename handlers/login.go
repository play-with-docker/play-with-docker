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
	"github.com/satori/go.uuid"
)

func LoggedInUser(rw http.ResponseWriter, req *http.Request) {
	cookie, err := ReadCookie(req)
	if err != nil {
		log.Println("Cannot read cookie")
		rw.WriteHeader(http.StatusUnauthorized)
		return
	}
	user, err := core.UserGet(cookie.Id)
	if err != nil {
		log.Printf("Couldn't get user with id %s. Got: %v\n", cookie.Id, err)
		rw.WriteHeader(http.StatusUnauthorized)
		return
	}
	json.NewEncoder(rw).Encode(user)
}

func ListProviders(rw http.ResponseWriter, req *http.Request) {
	playground := core.PlaygroundFindByDomain(req.Host)
	if playground == nil {
		log.Printf("Playground for domain %s was not found!", req.Host)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	providers := []string{}
	for name, _ := range config.Providers[playground.Id] {
		providers = append(providers, name)
	}
	json.NewEncoder(rw).Encode(providers)
}

func Login(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	providerName := vars["provider"]
	playground := core.PlaygroundFindByDomain(req.Host)
	if playground == nil {
		log.Printf("Playground for domain %s was not found!", req.Host)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	provider, found := config.Providers[playground.Id][providerName]
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
	if req.TLS != nil {
		scheme = "https"
	}
	host := "localhost"
	if req.Host != "" {
		host = req.Host
	}
	provider.RedirectURL = fmt.Sprintf("%s://%s/oauth/providers/%s/callback", scheme, host, providerName)
	url := provider.AuthCodeURL(loginRequest.Id, oauth2.SetAuthURLParam("nonce", uuid.NewV4().String()))

	http.Redirect(rw, req, url, http.StatusFound)
}

func LoginCallback(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	providerName := vars["provider"]
	playground := core.PlaygroundFindByDomain(req.Host)
	if playground == nil {
		log.Printf("Playground for domain %s was not found!", req.Host)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	provider, found := config.Providers[playground.Id][providerName]
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
	} else if providerName == "docker" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: tok.AccessToken},
		)
		tc := oauth2.NewClient(ctx, ts)
		resp, err := tc.Get("https://id.docker.com/api/id/v1/openid/userinfo")
		if err != nil {
			log.Printf("Could not get user from docker. Got: %v\n", err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		userInfo := map[string]string{}
		if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
			log.Printf("Could not decode user info. Got: %v\n", err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		user.ProviderUserId = userInfo["sub"]
		user.Name = userInfo["preferred_username"]
		user.Email = userInfo["email"]
		// Since DockerID doesn't return a user avatar, we try with twitter through avatars.io
		// Worst case we get a generic avatar
		user.Avatar = fmt.Sprintf("https://avatars.io/twitter/%s", user.Name)
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

	fmt.Fprintf(rw, `
<html>
    <head>
	<script>
	if (window.opener && !window.opener.closed) {
	    try {
	      window.opener.postMessage('done','*')
	    }
	    catch(e) {  }
	    window.close();
	}
	</script>
    </head>
    <body>
    </body>
</html>`)
}
