#!/bin/bash
rm certs/server/*

openssl ecparam -genkey -name secp384r1 -out certs/server/server.key
openssl req -new -key certs/server/server.key -out certs/server/server.csr
openssl x509 -req -in certs/server/server.csr -CA certs/ca/ca.crt -CAkey certs/ca/ca.key -CAcreateserial -out certs/server/server.crt -days 365 -sha256

rm certs/server/server.csr

openssl verify -CAfile certs/ca/ca.crt certs/server/server.crt
