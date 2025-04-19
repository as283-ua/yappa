package test

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/quic-go/qlog"
)

func GetHttp3Client(certPath, certificateOwner, caCertPath string) *http.Client {
	rootCAs := x509.NewCertPool()

	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		log.Fatal("Failed to read root CA certificate:", err)
	}

	rootCAs.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		RootCAs:    rootCAs,
		NextProtos: []string{"h3"},
	}

	if certificateOwner != "" {
		cert, err := tls.LoadX509KeyPair(certPath+"/"+certificateOwner+"/"+certificateOwner+".crt", certPath+"/"+certificateOwner+"/"+certificateOwner+".key")
		if err != nil {
			log.Fatal(err)
		}
		tlsConfig.Certificates = append(tlsConfig.Certificates, cert)
	}

	transport := &http3.Transport{
		TLSClientConfig: tlsConfig,
		QUICConfig: &quic.Config{
			MaxIdleTimeout:  60 * time.Second,
			EnableDatagrams: true,
			Tracer:          qlog.DefaultConnectionTracer,
		},
		EnableDatagrams: true,
	}

	return &http.Client{
		Transport: transport,
	}
}

func Encrypt(data, key []byte) (out []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, data, nil)
	out = append(nonce, ciphertext...)
	return out, nil
}

func Decrypt(data, key []byte) (out []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	out, err = gcm.Open(nil, nonce, ciphertext, nil)
	return
}
