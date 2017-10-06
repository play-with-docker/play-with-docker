package config

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/gorilla/securecookie"

	"golang.org/x/oauth2"
	oauth2FB "golang.org/x/oauth2/facebook"
	oauth2Github "golang.org/x/oauth2/github"
)

const (
	PWDHostnameRegex      = "[0-9]{1,3}-[0-9]{1,3}-[0-9]{1,3}-[0-9]{1,3}"
	PortRegex             = "[0-9]{1,5}"
	AliasnameRegex        = "[0-9|a-z|A-Z|-]*"
	AliasSessionRegex     = "[0-9|a-z|A-Z]{8}"
	AliasGroupRegex       = "(" + AliasnameRegex + ")-(" + AliasSessionRegex + ")"
	PWDHostPortGroupRegex = "^.*ip(" + PWDHostnameRegex + ")(?:-?(" + PortRegex + "))?(?:\\..*)?$"
	AliasPortGroupRegex   = "^.*pwd" + AliasGroupRegex + "(?:-?(" + PortRegex + "))?\\..*$"
)

var NameFilter = regexp.MustCompile(PWDHostPortGroupRegex)
var AliasFilter = regexp.MustCompile(AliasPortGroupRegex)

var PortNumber, Key, Cert, SessionsFile, PWDContainerName, L2ContainerName, L2Subdomain, PWDCName, HashKey, SSHKeyPath, L2RouterIP, DindVolumeSize, CookieHashKey, CookieBlockKey string
var UseLetsEncrypt, ExternalDindVolume, NoWindows bool
var LetsEncryptCertsDir string
var LetsEncryptDomains stringslice
var MaxLoadAvg float64
var ForceTLS bool
var Providers map[string]*oauth2.Config
var SecureCookie *securecookie.SecureCookie

var GithubClientID, GithubClientSecret string
var FacebookClientID, FacebookClientSecret string

type stringslice []string

func (i *stringslice) String() string {
	return fmt.Sprintf("%s", *i)
}
func (i *stringslice) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func ParseFlags() {
	flag.Var(&LetsEncryptDomains, "letsencrypt-domain", "List of domains to validate with let's encrypt")
	flag.StringVar(&LetsEncryptCertsDir, "letsencrypt-certs-dir", "/certs", "Path where let's encrypt certs will be stored")
	flag.BoolVar(&UseLetsEncrypt, "letsencrypt-enable", false, "Enabled let's encrypt tls certificates")
	flag.BoolVar(&ForceTLS, "tls", false, "Use TLS to connect to docker daemons")
	flag.StringVar(&PortNumber, "port", "3000", "Port number")
	flag.StringVar(&Key, "tls-server-key", "./pwd/server-key.pem", "Server key for SSL")
	flag.StringVar(&Cert, "tls-cert", "./pwd/server.pem", "Give a SSL cert")
	flag.StringVar(&SessionsFile, "save", "./pwd/sessions", "Tell where to store sessions file")
	flag.StringVar(&PWDContainerName, "name", "pwd", "Container name used to run PWD (used to be able to connect it to the networks it creates)")
	flag.StringVar(&L2ContainerName, "l2", "l2", "Container name used to run L2 Router")
	flag.StringVar(&L2RouterIP, "l2-ip", "", "Host IP address for L2 router ping response")
	flag.StringVar(&L2Subdomain, "l2-subdomain", "direct", "Subdomain to the L2 Router")
	flag.StringVar(&PWDCName, "cname", "", "CNAME given to this host")
	flag.StringVar(&HashKey, "hash_key", "salmonrosado", "Hash key to use for cookies")
	flag.StringVar(&DindVolumeSize, "dind-volume-size", "5G", "Dind volume folder size")
	flag.BoolVar(&NoWindows, "win-disable", false, "Disable windows instances")
	flag.BoolVar(&ExternalDindVolume, "dind-external-volume", false, "Use external dind volume though XFS volume driver")
	flag.Float64Var(&MaxLoadAvg, "maxload", 100, "Maximum allowed load average before failing ping requests")
	flag.StringVar(&SSHKeyPath, "ssh_key_path", "", "SSH Private Key to use")
	flag.StringVar(&CookieHashKey, "cookie-hash-key", "", "Hash key to use to validate cookies")
	flag.StringVar(&CookieBlockKey, "cookie-block-key", "", "Block key to use to encrypt cookies")

	flag.StringVar(&GithubClientID, "oauth-github-client-id", "", "Github OAuth Client ID")
	flag.StringVar(&GithubClientSecret, "oauth-github-client-secret", "", "Github OAuth Client Secret")

	flag.StringVar(&FacebookClientID, "oauth-facebook-client-id", "", "Facebook OAuth Client ID")
	flag.StringVar(&FacebookClientSecret, "oauth-facebook-client-secret", "", "Facebook OAuth Client Secret")

	flag.Parse()

	SecureCookie = securecookie.New([]byte(CookieHashKey), []byte(CookieBlockKey))

	registerOAuthProviders()
}

func registerOAuthProviders() {
	Providers = map[string]*oauth2.Config{}
	if GithubClientID != "" && GithubClientSecret != "" {
		conf := &oauth2.Config{
			ClientID:     GithubClientID,
			ClientSecret: GithubClientSecret,
			Scopes:       []string{"user:email"},
			Endpoint:     oauth2Github.Endpoint,
		}

		Providers["github"] = conf
	}
	if FacebookClientID != "" && FacebookClientSecret != "" {
		conf := &oauth2.Config{
			ClientID:     FacebookClientID,
			ClientSecret: FacebookClientSecret,
			Scopes:       []string{"email", "public_profile"},
			Endpoint:     oauth2FB.Endpoint,
		}

		Providers["facebook"] = conf
	}
}

func GetDindImageName() string {
	dindImage := os.Getenv("DIND_IMAGE")
	defaultDindImageName := "franela/dind"
	if len(dindImage) == 0 {
		dindImage = defaultDindImageName
	}
	return dindImage
}

func GetSSHImage() string {
	sshImage := os.Getenv("SSH_IMAGE")
	defaultSSHImage := "franela/ssh"
	if len(sshImage) == 0 {
		return defaultSSHImage
	}
	return sshImage
}

func GetDuration(reqDur string) time.Duration {
	var defaultDuration = 4 * time.Hour
	if reqDur != "" {
		if dur, err := time.ParseDuration(reqDur); err == nil && dur <= defaultDuration {
			return dur
		}
		return defaultDuration
	}

	envDur := os.Getenv("EXPIRY")
	if dur, err := time.ParseDuration(envDur); err == nil {
		return dur
	}

	return defaultDuration
}
