package middleware

import (
	"log"
	"math/big"
	"net/http"
)

func IsChatServer(serialNumber *big.Int, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		isServer := false

		for _, cert := range req.TLS.PeerCertificates {
			if serialNumber.Cmp(cert.SerialNumber) == 0 {
				isServer = true
				break
			}
		}

		if !isServer {
			log.Printf("Unauthorized access to restricted end-point by %v\n", req.RemoteAddr)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, req)
	})
}
