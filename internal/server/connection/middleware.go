package connection

import (
	"crypto/x509"
	"net/http"
)

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
