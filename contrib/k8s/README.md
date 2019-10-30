# Kubernetes manifests

This is a set of kubernetes manifests to seed a local minikube cluster with grafain.
Please note that the order of files matters. See `seed-cluster.sh`.

The `grafain-hook.yaml` must not be applied before the pod is ready. Otherwise the grafain startup may be blocked on 
expecting grafain to answer admission requests already.
