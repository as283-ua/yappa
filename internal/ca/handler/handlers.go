package handler

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"io"
	"net/http"
	"sync"
)

var allowedUsers map[string][]byte = make(map[string][]byte)
var mu sync.Mutex

func AllowUser(w http.ResponseWriter, req *http.Request) {
	username := req.PathValue("username")
	token := make([]byte, req.ContentLength)

	_, err := req.Body.Read(token)
	if err != nil && err != io.EOF {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	mu.Lock()
	allowedUsers[username] = token
	mu.Unlock()

	w.WriteHeader(200)
}

func SignCert(w http.ResponseWriter, req *http.Request) {
	username := req.PathValue("username")

	var tokenAllowed = make([]byte, 0)

	mu.Lock()
	token, ok := allowedUsers[username]
	mu.Unlock()
	if !ok || !bytes.Equal(token, tokenAllowed) {
		http.Error(w, "User not allowed", http.StatusUnauthorized)
		return
	}

	csrBytes := make([]byte, req.ContentLength)
	_, err := req.Body.Read(csrBytes)
	if err != nil {
		http.Error(w, "Failed to read CSR", http.StatusBadRequest)
		return
	}

	block, _ := pem.Decode(csrBytes)
	if block == nil || block.Type != "CERTIFICATE REQUEST" {
		http.Error(w, "Invalid CSR", http.StatusBadRequest)
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

	w.Write([]byte("Signed cert goes here"))
	w.WriteHeader(200)
	w.Write([]byte("Work in progress"))
}

func Getcertificates(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte("Work in progress"))
}

func Revoke(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte("Work in progress"))
}

func Reinstate(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte("Work in progress"))
}
