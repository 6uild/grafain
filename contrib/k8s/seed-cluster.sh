#!/usr/bin/env bash

set -eox pipefail

kubectl apply -f genesis.yaml
kubectl apply -f grafain-rbac.yaml
kubectl apply -f grafain-secret.yaml
kubectl apply -f grafain.yaml
# wait for

while [ "$(kubectl get pod grafain-0 -o jsonpath='{.status.containerStatuses[1].ready}')" != "true" ]; do
    sleep 5
done

kubectl apply -f grafain-hook.yaml
