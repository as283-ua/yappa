package middleware

import (
	"bytes"
	"crypto/sha256"
	"net/http"
)

func IsChatServer(serverCertHash []byte, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		isServer := false

		for _, cert := range req.TLS.PeerCertificates {
			hash := sha256.Sum256(cert.Raw)
			if bytes.Equal(serverCertHash, hash[:]) {
				isServer = true
				break
			}
		}

		if !isServer {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}

		next.ServeHTTP(w, req)
	})
}
