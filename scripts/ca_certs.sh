#!/bin/bash
rm certs/ca/ca.key
rm certs/ca/ca.crt

mkdir -p certs/ca

openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:4096 -out certs/ca/ca.key
openssl req -x509 -new -key certs/ca/ca.key -sha256 -days 3650 -out certs/ca/ca.crt -config certs/ca.cnf -extensions v3_ca
