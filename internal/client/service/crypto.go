package service

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"log"
)

type PrivKeyBundle struct {
	Pem []byte
	Key *ecdsa.PrivateKey
}

func GeneratePrivKey() (*PrivKeyBundle, error) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Println("Crypto error:", err)
		return nil, errors.New("crypto error")
	}

	privKeyBytes, err := x509.MarshalECPrivateKey(privKey)
	if err != nil {
		log.Println("x509 error:", err)
		return nil, errors.New("crypto error")
	}

	privKeyPem := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privKeyBytes,
	})

	if privKeyPem == nil {
		return nil, errors.New("crypto error")
	}

	return &PrivKeyBundle{Pem: privKeyPem, Key: privKey}, nil
}

func GenerateCSR(privateKey *ecdsa.PrivateKey, username string) ([]byte, error) {
	template := x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: username,
		},
	}

	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, &template, privateKey)
	if err != nil {
		return nil, err
	}

	csrPem := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrBytes,
	})

	return csrPem, nil
}
