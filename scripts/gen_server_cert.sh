#!/bin/bash

if [ "$#" -ne 1 ]; then
    echo "Error: Exactly 1 argument is required."
    echo "Usage: $0 <argument>"
    exit 1
fi

mkdir -p certs/$1
rm -rf certs/$1/*

openssl ecparam -genkey -name secp384r1 | openssl pkcs8 -topk8 -nocrypt -out certs/$1/$1.key
openssl req -new -key certs/$1/$1.key -out certs/$1/$1.csr -config certs/server.cnf

openssl req -noout -text -in certs/$1/$1.csr | grep -A 1 "Subject Alternative Name"

openssl x509 -req -days 365 -in certs/$1/$1.csr -CA certs/ca/ca.crt -CAkey certs/ca/ca.key -CAcreateserial -out certs/$1/$1.crt -extensions req_ext -extfile certs/server.cnf

rm certs/$1/$1.csr

openssl verify -CAfile certs/ca/ca.crt certs/$1/$1.crt
