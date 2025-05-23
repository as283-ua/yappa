package signature

import (
	"math/big"
	"net/http"

	"github.com/as283-ua/yappa/internal/ca/logging"
)

var log = logging.GetLogger()

func MatchCertSerialNumber(serialNumber *big.Int, next http.Handler) http.Handler {
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
