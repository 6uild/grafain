#!/bin/bash
set -eox pipefail

#cfssl gencert -initca ca-csr.json | cfssljson -bare ca

cfssl gencert \
  -ca=ca.pem \
  -ca-key=ca-key.pem \
  -config=ca-config.json \
  -hostname=grafain.default.svc \
  -profile=default \
  local-csr.json | cfssljson -bare local

# rename for local testing
mv local.pem tls.crt
mv local-key.pem tls.key

cfssl gencert \
  -ca=ca.pem \
  -ca-key=ca-key.pem \
  -config=ca-config.json \
  -hostname=grafain.default.svc \
  -profile=default \
  grafain-csr.json | cfssljson -bare grafain

# upload to cluster
kubectl create secret tls tls-grafain \
  --cert=grafain.pem \
  --key=grafain-key.pem

printf "Encoded cert for ca-bundle:\n"
base64 --input=ca.pem