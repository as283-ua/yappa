package connection

import (
	"context"
	"crypto/ecdh"
	"crypto/x509"
	"encoding/base64"
	"net/http"

	"github.com/as283-ua/yappa/internal/server/logging"
	"golang.org/x/crypto/curve25519"
)

const EcdhCtxKey = "ecdh_pubkey"

func RequireCertificate(tlsVerifyOpts x509.VerifyOptions, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(r.TLS.PeerCertificates) == 0 {
			http.Error(w, "No valid certificates provided", http.StatusBadRequest)
			return
		}

		if _, err := r.TLS.PeerCertificates[0].Verify(tlsVerifyOpts); err != nil {
			http.Error(w, "No valid certificates provided", http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireEcdh(next http.Handler) http.Handler {
	logger := logging.GetLogger()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ecdh64 := r.Header.Get("X-Ecdh")
		if ecdh64 == "" {
			http.Error(w, "X-Ecdh is a mandatory header", http.StatusBadRequest)
			return
		}

		ecdhKeyBytes, err := base64.StdEncoding.DecodeString(ecdh64)
		if err != nil {
			logger.Println(err)
			http.Error(w, "Invalid format", http.StatusBadRequest)
			return
		}
		if len(ecdhKeyBytes) != curve25519.PointSize {
			http.Error(w, "Invalid key length", http.StatusBadRequest)
			return
		}
		ecdhKey, err := ecdh.X25519().NewPublicKey(ecdhKeyBytes)
		if err != nil {
			logger.Println(err)
			http.Error(w, "Invalid format", http.StatusBadRequest)
			return
		}
		ctx := context.WithValue(r.Context(), EcdhCtxKey, ecdhKey)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
