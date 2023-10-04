#!/bin/sh

set -e

ca="127.0.0.1-ca.crt"
key="127.0.0.1-key.pem"
cert="127.0.0.1-cert.pem"
csr="127.0.0.1.csr"
days="3650"

# Generate a certificate authority key, store only in memory.
rootca="$(openssl genrsa 2048)"

# Self-sign rootca, store cert as file.
echo "$rootca" | openssl req -x509 -new -nodes -sha512 -days "$days" \
    -config ca.cnf -key /dev/stdin -out "$ca"

# Create key.
openssl genrsa -out "$key" 2048

# Create certificate sign request.
openssl req -new -config server.cnf -key "$key" -out "$csr"

# Sign with rootca.
echo "$rootca" | openssl x509 -req -days "$days" -sha512 -CAcreateserial \
    -extensions req_v3_usr -extfile server.cnf \
    -in "$csr" -CA "$ca" -CAkey /dev/stdin -out "$cert"

# Turn the certificate into a proper chain.
cat "$ca" >>"$cert"

rm *.srl *.csr

unset rootca
