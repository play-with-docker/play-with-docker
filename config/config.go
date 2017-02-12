package config

import "flag"

var SSLPortNumber, PortNumber, Key, Cert string

func ParseFlags() {
	flag.StringVar(&PortNumber, "port", "3000", "Give a TCP port to run the application")
	flag.StringVar(&SSLPortNumber, "sslPort", "3001", "Give a SSL TCP port")
	flag.StringVar(&Key, "key", "./pwd/server-key.pem", "Server key for SSL")
	flag.StringVar(&Cert, "cert", "./pwd/server.pem", "Give a SSL cert")
	flag.Parse()
}
