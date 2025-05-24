package signature

import (
	"bytes"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/as283-ua/yappa/api/gen/ca"
	"github.com/as283-ua/yappa/api/gen/server"
	"github.com/as283-ua/yappa/internal/ca/logging"
	"google.golang.org/protobuf/proto"
)

type RegTokens struct {
	certificationToken []byte
	confirmationToken  []byte
}

var allowedUsers map[string]RegTokens = make(map[string]RegTokens)
var mu sync.Mutex

func validateAllow(allow *ca.AllowUser) error {
	if len(allow.Token) != 64 {
		return errors.New("invalid token")
	}

	if len(allow.User) < 3 {
		return errors.New("invalid username length")
	}

	return nil
}

func AllowUser(w http.ResponseWriter, req *http.Request) {
	log := logging.GetLogger()
	allowUser := &ca.AllowUser{}

	content, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	proto.Unmarshal(content, allowUser)

	err = validateAllow(allowUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	token := make([]byte, 64)
	rand.Read(token)

	confirm := &server.ConfirmRegistrationToken{
		User:  allowUser.User,
		Token: token,
	}

	confirmBytes, err := proto.Marshal(confirm)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Println("Proto marshal error: " + err.Error())
		return
	}

	mu.Lock()
	allowedUsers[allowUser.User] = RegTokens{certificationToken: allowUser.Token, confirmationToken: token}
	mu.Unlock()

	w.WriteHeader(http.StatusOK)
	w.Write(confirmBytes)

	log.Println("Allowed user " + allowUser.User)
}

func SignCert(caCert *x509.Certificate, caKey any) func(w http.ResponseWriter, req *http.Request) {
	log := logging.GetLogger()
	return func(w http.ResponseWriter, req *http.Request) {
		certRequest := &ca.CertRequest{}
		body, err := io.ReadAll(req.Body)

		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			log.Println("Internal error: " + err.Error())
			return
		}

		proto.Unmarshal(body, certRequest)

		mu.Lock()
		token, ok := allowedUsers[certRequest.User]
		mu.Unlock()

		if !ok || !bytes.Equal(token.certificationToken, certRequest.Token) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		csrBytes := certRequest.Csr

		block, _ := pem.Decode(csrBytes)
		if block == nil || block.Type != "CERTIFICATE REQUEST" {
			http.Error(w, "Invalid CSR", http.StatusBadRequest)
			log.Println("Invalid CSR: " + string(csrBytes))
			return
		}

		csr, err := x509.ParseCertificateRequest(block.Bytes)
		if err != nil {
			http.Error(w, "Failed to parse CSR", http.StatusBadRequest)
			return
		}

		if err := csr.CheckSignature(); err != nil {
			http.Error(w, "Invalid CSR signature", http.StatusBadRequest)
			return
		}

		if csr.Subject.CommonName != certRequest.User {
			http.Error(w, "Invalid CSR common name", http.StatusBadRequest)
			return
		}

		template := &x509.Certificate{
			SerialNumber:          big.NewInt(time.Now().UnixNano()),
			Subject:               csr.Subject,
			NotBefore:             time.Now(),
			NotAfter:              time.Now().Add(365 * 24 * time.Hour),
			KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
			ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			BasicConstraintsValid: true,
		}

		signedCert, err := x509.CreateCertificate(rand.Reader, template, caCert, csr.PublicKey, caKey)
		if err != nil {
			http.Error(w, "Failed to sign certificate", http.StatusInternalServerError)
			log.Println("Internal error: " + err.Error())
			return
		}

		signedCertPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: signedCert,
		})

		w.Header().Add("Content-Type", "application/x-protobuf")

		cert := &ca.CertResponse{
			Cert:  signedCertPEM,
			Token: token.confirmationToken,
		}

		bytes, err := proto.Marshal(cert)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			log.Println("Internal error: " + err.Error())
			return
		}

		w.Write(bytes)
		w.WriteHeader(http.StatusOK)

		log.Println("Signed certificate for user " + certRequest.User)

		mu.Lock()
		delete(allowedUsers, certRequest.User)
		mu.Unlock()
	}
}

func Getcertificates(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("Work in progress"))
}

func Revoke(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("Work in progress"))
}

func Reinstate(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("Work in progress"))
}
