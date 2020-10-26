#!/bin/bash

set -eux

prefix="$1"

cat >/tmp/openssl.conf <<EOF
[req]
default_bits  = 2048
distinguished_name = req_distinguished_name
req_extensions = req_ext
x509_extensions = v3_req
prompt = no

[req_distinguished_name]
countryName = XX
stateOrProvinceName = N/A
localityName = N/A
organizationName = N/A
commonName = N/A

[req_ext]
subjectAltName = @alt_names

[v3_req]
subjectAltName = @alt_names

[alt_names]
DNS.1 = determined-master-ci
EOF

openssl req -new -x509 -sha256 -newkey rsa:2048 -nodes -days 365 \
  -config /tmp/openssl.conf \
  -out "$prefix".crt \
  -keyout "$prefix".key
