package config

import "flag"

var SSLPortNumber, PortNumber, Key, Cert, SessionsFile, PWDContainerName string
var MaxLoadAvg float64

func ParseFlags() {
	flag.StringVar(&PortNumber, "port", "3000", "Give a TCP port to run the application")
	flag.StringVar(&SSLPortNumber, "sslPort", "3001", "Give a SSL TCP port")
	flag.StringVar(&Key, "key", "./pwd/server-key.pem", "Server key for SSL")
	flag.StringVar(&Cert, "cert", "./pwd/server.pem", "Give a SSL cert")
	flag.StringVar(&SessionsFile, "save", "./pwd/sessions", "Tell where to store sessions file")
	flag.StringVar(&PWDContainerName, "name", "pwd", "Container name used to run PWD (used to be able to connect it to the networks it creates)")
	flag.Float64Var(&MaxLoadAvg, "maxload", 100, "Maximum allowed load average before failing ping requests")
	flag.Parse()
}
