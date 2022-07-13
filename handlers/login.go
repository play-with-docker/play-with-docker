package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/oauth2"

	"github.com/google/go-github/github"
	"github.com/gorilla/mux"
	"github.com/play-with-docker/play-with-docker/config"
	"github.com/play-with-docker/play-with-docker/pwd/types"
	uuid "github.com/satori/go.uuid"
	"google.golang.org/api/option"
	"google.golang.org/api/people/v1"
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
	for name := range config.Providers[playground.Id] {
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

	if playground.AuthRedirectBase != "" {
		provider.RedirectURL = fmt.Sprintf("%s/oauth/providers/%s/callback", playground.AuthRedirectBase, providerName)
	} else {
		scheme := "http"
		if req.TLS != nil {
			scheme = "https"
		}
		host := "localhost"
		if req.Host != "" {
			host = req.Host
		}
		provider.RedirectURL = fmt.Sprintf("%s://%s/oauth/providers/%s/callback", scheme, host, providerName)
	}

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
	} else if providerName == "google" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: tok.AccessToken},
		)
		tc := oauth2.NewClient(ctx, ts)

		p, err := people.NewService(ctx, option.WithHTTPClient(tc))
		if err != nil {
			log.Printf("Could not initialize people service . Got: %v\n", err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		person, err := p.People.Get("people/me").PersonFields("emailAddresses,names").Do()
		if err != nil {
			log.Printf("Could not initialize people service . Got: %v\n", err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		user.Email = person.EmailAddresses[0].Value
		user.Name = person.Names[0].GivenName
		user.ProviderUserId = person.ResourceName

	} else if providerName == "docker" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: tok.AccessToken},
		)
		tc := oauth2.NewClient(ctx, ts)

		endpoint := getDockerEndpoint(playground)
		resp, err := tc.Get(fmt.Sprintf("https://%s/userinfo", endpoint))
		if err != nil {
			log.Printf("Could not get user from docker. Got: %v\n", err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		userInfo := map[string]interface{}{}
		if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
			log.Printf("Could not decode user info. Got: %v\n", err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		user.ProviderUserId = strings.Split(userInfo["sub"].(string), "|")[1]
		user.Name = userInfo["https://hub.docker.com"].(map[string]interface{})["username"].(string)
		user.Email = userInfo["https://hub.docker.com"].(map[string]interface{})["email"].(string)
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

	cookieData := CookieID{Id: user.Id, UserName: user.Name, UserAvatar: user.Avatar, ProviderId: user.ProviderUserId}

	host := "localhost"
	if req.Host != "" {
		// we get the parent domain so cookie is set
		// in all subdomain and siblings
		host = getParentDomain(req.Host)
	}

	if err := cookieData.SetCookie(rw, host); err != nil {
		log.Printf("Could not encode cookie. Got: %v\n", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	r, _ := playground.Extras.GetString("LoginRedirect")

	fmt.Fprintf(rw, `
<html>
    <head>
	<script>
	if (window.opener && !window.opener.closed) {
	    try {
	      window.opener.postMessage('done','*');
	    }
	    catch(e) {  }
	    window.close();
	} else {
	    window.location = '%s';
	}
	</script>
    </head>
    <body>
    </body>
</html>`, r)
}

// getParentDomain returns the parent domain (if available)
// of the currend domain
func getParentDomain(host string) string {
	levels := strings.Split(host, ".")
	if len(levels) > 2 {
		return strings.Join(levels[1:], ".")
	}
	return host
}

func getDockerEndpoint(p *types.Playground) string {
	if len(p.DockerHost) > 0 {
		return p.DockerHost
	}
	return "login.docker.com"
}
