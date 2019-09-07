#!/bin/bash

cfssl gencert -initca ca-csr.json | cfssljson -bare ca
cfssl gencert \
  -ca=ca.pem \
  -ca-key=ca-key.pem \
  -config=ca-config.json \
  -hostname=grafain.default.svc \
  -profile=default \
  grafain-csr.json | cfssljson -bare grafain

kubectl create secret tls tls-grafain \
  --cert=grafain.pem \
  --key=grafain-key.pem

printf "Encoded cert for ca-bundle:\n"
base64 --input=ca.pem