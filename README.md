## Grafain

Grafain is a kubernetes policy and permission server server. It receive requests from the 
[admission controller](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/) via webhooks
and returns decisions based on internal rules.

## Quickstart with [Minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/)

* `minikube start`                  - start environment
* `kubectl apply -f contrib/k8s`    - deploy grafain components
* `kubectl get pods`                - check grafain pod is running
* `kubeclt logs -f grafain-0`       - watch log
* `kubectl create deployment microbot --image=dontrebootme/microbot:v1` - deploy a random pod


## How to build a new docker artifact

```sh
make dist
```

## Other Admission Controller
* https://github.com/IBM/portieris
* https://github.com/open-policy-agent/gatekeeper
* https://github.com/grafeas/kritis
## Other Resources
* [ValidatingWebhookConfiguration](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers)
* [A grafeas-tutorial](https://github.com/kelseyhightower/grafeas-tutorial)

## License
TBD