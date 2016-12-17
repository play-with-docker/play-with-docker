package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/docker/machine/libmachine/auth"
	"github.com/docker/machine/libmachine/cert"
)

func main() {
	if err := generateCerts(); err != nil {
		log.Fatal(err)
	}

	log.Println("Certs generated successfully")
}

func generateCerts() error {
	certsDir := "."
	opts := &auth.Options{
		CertDir:          certsDir,
		CaCertPath:       filepath.Join(certsDir, "ca.pem"),
		CaPrivateKeyPath: filepath.Join(certsDir, "ca-key.pem"),
		ClientCertPath:   filepath.Join(certsDir, "cert.pem"),
		ClientKeyPath:    filepath.Join(certsDir, "key.pem"),
		ServerCertPath:   filepath.Join(certsDir, "server.pem"),
		ServerKeyPath:    filepath.Join(certsDir, "server-key.pem"),
	}

	// Generate client cert
	if err := cert.BootstrapCertificates(opts); err != nil {
		log.Println("Error bootstrapping client certificates")
		return err
	}

	hosts := []string{"*.play-with-docker.com", "*.localhost"}
	// Generate server cert
	err := cert.GenerateCert(&cert.Options{
		Hosts:       hosts,
		CertFile:    opts.ServerCertPath,
		KeyFile:     opts.ServerKeyPath,
		CAFile:      opts.CaCertPath,
		CAKeyFile:   opts.CaPrivateKeyPath,
		Org:         "play-with-docker.com",
		Bits:        2048,
		SwarmMaster: false,
	})

	if err != nil {
		return fmt.Errorf("error generating server cert: %s", err)
	}
	return nil
}
