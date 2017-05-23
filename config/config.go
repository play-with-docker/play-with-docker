package config

import (
	"flag"
	"os"
	"regexp"
	"time"
)

const (
	PWDHostnameRegex      = "[0-9]{1,3}-[0-9]{1,3}-[0-9]{1,3}-[0-9]{1,3}"
	PortRegex             = "[0-9]{1,5}"
	AliasnameRegex        = "[0-9|a-z|A-Z|-]*"
	AliasSessionRegex     = "[0-9|a-z|A-Z]{8}"
	AliasGroupRegex       = "(" + AliasnameRegex + ")-(" + AliasSessionRegex + ")"
	PWDHostPortGroupRegex = "^.*pwd(" + PWDHostnameRegex + ")(?:-?(" + PortRegex + "))?\\..*$"
	AliasPortGroupRegex   = "^.*pwd" + AliasGroupRegex + "(?:-?(" + PortRegex + "))?\\..*$"
)

var NameFilter = regexp.MustCompile(PWDHostPortGroupRegex)
var AliasFilter = regexp.MustCompile(AliasPortGroupRegex)

var SSLPortNumber, PortNumber, Key, Cert, SessionsFile, PWDContainerName, PWDCName, HashKey string
var MaxLoadAvg float64

func ParseFlags() {
	flag.StringVar(&PortNumber, "port", "3000", "Give a TCP port to run the application")
	flag.StringVar(&SSLPortNumber, "sslPort", "3001", "Give a SSL TCP port")
	flag.StringVar(&Key, "key", "./pwd/server-key.pem", "Server key for SSL")
	flag.StringVar(&Cert, "cert", "./pwd/server.pem", "Give a SSL cert")
	flag.StringVar(&SessionsFile, "save", "./pwd/sessions", "Tell where to store sessions file")
	flag.StringVar(&PWDContainerName, "name", "pwd", "Container name used to run PWD (used to be able to connect it to the networks it creates)")
	flag.StringVar(&PWDCName, "cname", "host1", "CNAME given to this host")
	flag.StringVar(&HashKey, "hash_key", "salmonrosado", "Hash key to use for cookies")
	flag.Float64Var(&MaxLoadAvg, "maxload", 100, "Maximum allowed load average before failing ping requests")
	flag.Parse()
}
func GetDindImageName() string {
	dindImage := os.Getenv("DIND_IMAGE")
	defaultDindImageName := "franela/dind"
	if len(dindImage) == 0 {
		dindImage = defaultDindImageName
	}
	return dindImage
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
