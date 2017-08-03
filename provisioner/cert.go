package provisioner

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"time"
)

func newCertificate(org string) (*x509.Certificate, error) {
	now := time.Now()
	// need to set notBefore slightly in the past to account for time
	// skew in the VMs otherwise the certs sometimes are not yet valid
	notBefore := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute()-5, 0, 0, time.Local)
	notAfter := notBefore.Add(time.Hour * 24 * 1080)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}

	return &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{org},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement,
		BasicConstraintsValid: true,
	}, nil

}

func GenerateCACertificate(org string) ([]byte, []byte, error) {
	template, err := newCertificate(org)
	if err != nil {
		return nil, nil, err
	}

	template.IsCA = true
	template.KeyUsage |= x509.KeyUsageCertSign
	template.KeyUsage |= x509.KeyUsageKeyEncipherment
	template.KeyUsage |= x509.KeyUsageKeyAgreement

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, err
	}

	cert := new(bytes.Buffer)
	key := new(bytes.Buffer)
	pem.Encode(cert, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	pem.Encode(key, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	return cert.Bytes(), key.Bytes(), nil
}

func GenerateClientCertificate(org string, caCert, caKey []byte) ([]byte, []byte, error) {
	template, err := newCertificate(org)
	if err != nil {
		return nil, nil, err
	}

	template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	template.KeyUsage = x509.KeyUsageDigitalSignature

	tlsCert, err := tls.X509KeyPair(caCert, caKey)
	if err != nil {
		return nil, nil, err
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	x509Cert, err := x509.ParseCertificate(tlsCert.Certificate[0])
	if err != nil {
		return nil, nil, err
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, x509Cert, &priv.PublicKey, tlsCert.PrivateKey)
	if err != nil {
		return nil, nil, err
	}

	cert := new(bytes.Buffer)
	key := new(bytes.Buffer)

	pem.Encode(cert, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	pem.Encode(key, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	return cert.Bytes(), key.Bytes(), nil
}

func GenerateServerCertificate(org string, caCert, caKey []byte, hosts []string) ([]byte, []byte, error) {
	template, err := newCertificate(org)
	if err != nil {
		return nil, nil, err
	}

	template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	tlsCert, err := tls.X509KeyPair(caCert, caKey)
	if err != nil {
		return nil, nil, err
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	x509Cert, err := x509.ParseCertificate(tlsCert.Certificate[0])
	if err != nil {
		return nil, nil, err
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, x509Cert, &priv.PublicKey, tlsCert.PrivateKey)
	if err != nil {
		return nil, nil, err
	}

	cert := new(bytes.Buffer)
	key := new(bytes.Buffer)

	pem.Encode(cert, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	pem.Encode(key, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	return cert.Bytes(), key.Bytes(), nil
}
