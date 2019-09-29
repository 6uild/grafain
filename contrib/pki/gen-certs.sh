#!/bin/bash

cfssl gencert -initca ca-csr.json | cfssljson -bare ca
cfssl gencert \
  -ca=ca.pem \
  -ca-key=ca-key.pem \
  -config=ca-config.json \
  -hostname=grafain.default.svc \
  -profile=default \
  grafain-csr.json | cfssljson -bare grafain

mv grafain.pem tls.crt
mv grafain-key.pem tls.key

kubectl create secret tls tls-grafain \
  --cert=tls.crt \
  --key=tls.key

printf "Encoded cert for ca-bundle:\n"
base64 --input=ca.pem