package main

import (
	cryptoRand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"math/rand"
	"os"
	"time"
)

func BasicCert(random *rand.Rand) (*x509.Certificate, *rsa.PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(cryptoRand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	ca := &x509.Certificate{
		SerialNumber: big.NewInt(random.Int63()),
		Subject: pkix.Name{
			Country:            []string{"CN"},
			Organization:       []string{"Anonymous"},
			OrganizationalUnit: []string{"5L"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().AddDate(3, 0, 0),
	}

	return ca, privateKey, nil
}

func GenerateRootCert(random *rand.Rand) (*x509.Certificate, *rsa.PrivateKey, error) {
	cert, key, err := BasicCert(random)
	if err != nil {
		return nil, nil, err
	}

	cert.Subject.CommonName = "gobuilder-root"
	cert.BasicConstraintsValid = true
	cert.IsCA = true
	cert.KeyUsage = x509.KeyUsageCertSign

	return cert, key, nil
}

func GenerateServerCert(random *rand.Rand) (*x509.Certificate, *rsa.PrivateKey, error) {
	cert, key, err := BasicCert(random)
	if err != nil {
		return nil, nil, err
	}

	cert.Subject.CommonName = "gobuilder-server"
	cert.DNSNames = []string{"gobuilder-quic"}
	cert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	cert.KeyUsage = x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment

	return cert, key, nil
}

func GenerateClientCert(random *rand.Rand) (*x509.Certificate, *rsa.PrivateKey, error) {
	cert, key, err := BasicCert(random)
	if err != nil {
		return nil, nil, err
	}

	cert.Subject.CommonName = "gobuilder-client"
	cert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	cert.KeyUsage = x509.KeyUsageDigitalSignature

	return cert, key, nil
}

func SaveDataToPEM(data []byte, t, path string) error {
	o, err := os.Create(path)
	if err != nil {
		return err
	}
	defer o.Close()

	return pem.Encode(o, &pem.Block{
		Type:  t,
		Bytes: data,
	})
}

func SaveCert(ca *x509.Certificate, cert *x509.Certificate, certPrivateKey *rsa.PrivateKey, privateKey *rsa.PrivateKey, name string) error {
	der, err := x509.CreateCertificate(cryptoRand.Reader, cert, ca, &certPrivateKey.PublicKey, privateKey)
	if err != nil {
		return err
	}

	if err := SaveDataToPEM(der, "CERTIFICATE", name+".pem"); err != nil {
		return err
	}

	if err := SaveDataToPEM(x509.MarshalPKCS1PrivateKey(certPrivateKey),
		"RSA PRIVATE KEY", name+".key"); err != nil {
		return err
	}

	return nil
}

func GenerateCertAndKey() error {
	random := rand.New(rand.NewSource(time.Now().UnixMicro()))

	ca, caPrivateKey, err := GenerateRootCert(random)
	if err != nil {
		return err
	}
	if err := SaveCert(ca, ca, caPrivateKey, caPrivateKey, "gobuilder-root"); err != nil {
		return err
	}

	server, serverPrivateKey, err := GenerateServerCert(random)
	if err != nil {
		return err
	}
	if err := SaveCert(ca, server, serverPrivateKey, caPrivateKey, "gobuilder-server"); err != nil {
		return err
	}

	client, clientPrivateKey, err := GenerateClientCert(random)
	if err != nil {
		return err
	}
	if err := SaveCert(ca, client, clientPrivateKey, caPrivateKey, "gobuilder-client"); err != nil {
		return err
	}

	return nil
}
