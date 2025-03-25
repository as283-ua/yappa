#!/bin/bash
rm certs/ca/ca.key
rm certs/ca/ca.crt

mkdir -p certs/ca

openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-384 -out certs/ca/ca.key
openssl req -x509 -new -key certs/ca/ca.key -sha256 -days 3650 -out certs/ca/ca.crt -config certs/ca.cnf -extensions v3_ca
