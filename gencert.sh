#!/bin/bash
# generate test certificates for dex

set -e

echo "Generating Dex test certificates..."

WORKDIR=$(dirname "$0")
SSLDIR="${WORKDIR}/ssl"

# change this domain to match Dex issuer
DEX_DOMAIN="dex.127.0.0.1.nip.io"

mkdir -p "${SSLDIR}"
cd "${SSLDIR}"

# Generate CA private key
openssl genrsa -out ca.key 2048

# Generate CA certificate
cat > req.cnf <<EOF
[req]
distinguished_name = req_distinguished_name
x509_extensions = v3_ca
[req_distinguished_name]
[v3_ca]
subjectKeyIdentifier = hash
authorityKeyIdentifier = keyid:always,issuer
basicConstraints = critical,CA:true
keyUsage = critical, digitalSignature, cRLSign, keyCertSign
EOF

openssl req -new -x509 -days 365 -key ca.key -out ca.crt -subj "/CN=Dex Test CA" -config req.cnf

# Generate Dex server private key
openssl genrsa -out dex.key 2048

# Generate Dex server certificate
cat > dex.conf <<EOF
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
[req_distinguished_name]
[v3_req]
subjectAltName = @alt_names
[alt_names]
DNS.1 = ${DEX_DOMAIN}
DNS.2 = dex.dex
DNS.3 = dex.dex.svc
DNS.4 = dex.dex.svc.cluster.local
DNS.5 = localhost
IP.1 = 127.0.0.1
EOF

openssl req -new -key dex.key -out dex.csr -subj "/CN=${DEX_DOMAIN}" -config dex.conf

# Sign the server certificate with CA
openssl x509 -req -in dex.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out dex.crt -days 365 -extensions v3_req -extfile dex.conf

# Clean up CSR
rm -f dex.csr
