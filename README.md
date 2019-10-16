# Grafain

Grafain is a kubernetes policy and permission server server. It receive requests from the 
[admission controller](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/) via webhooks
and returns decisions based on internal rules.
## Server


### Quickstart with [Minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/)
```sh
minikube start`                  # start environment
kubectl apply -f contrib/k8s`    # deploy grafain components
kubectl get pods`                # check grafain pod is running
kubeclt logs -f grafain-0`       # watch log
kubectl create deployment microbot --image=dontrebootme/microbot:v1` # deploy a random pod -> should fail
```

## Client
The `grafaincli` is a commend line client to interact with the running grafaind server through the Blockchain engine. 
```sh
# build CLI client
go build ./cmd/grafaincli

# create a new private key
./grafaincli mnemonic | ./grafaincli keygen -key $(pwd)/my_grafain.key

# set endpoint address for the grafain cli
export GRAFAINCLI_TM_ADDR=$(minikube service grafain-rpc --url)

# add a new artifact to the system
./grafaincli create-artifact -image="foo/bar:any" -digest="anyValidDigest" \
    | ./grafaincli sign -key=$(pwd)/my_grafain.key \
    | ./grafaincli submit

# query all artifacts
./grafaincli query -path=/artifacts

# query by image
./grafaincli query -path=/artifacts/image -data foo/bar:any

# delete artifact by internal id (=key)
./grafaincli del-artifact -id=1 \
    | ./grafaincli sign -key=$(pwd)/my_grafain.key \
    | ./grafaincli submit
```

## Development
### How to build a new docker artifacts

```sh
make dist
```

## Manual test
```sh
curl -X POST -k  -H "Content-Type: application/json"  -d '
{
  "kind": "AdmissionReview",
  "apiVersion": "admission.k8s.io/v1beta1",
  "request": {
    "uid": "181988ef-db4e-4023-9af8-ea1121ccfa9a",
    "kind": {
      "group": "",
      "version": "v1",
      "kind": "Pod"
    },
    "resource": {
      "group": "",
      "version": "v1",
      "resource": "pods"
    },
    "requestKind": {
      "group": "",
      "version": "v1",
      "kind": "Pod"
    },
    "requestResource": {
      "group": "",
      "version": "v1",
      "resource": "pods"
    },
    "name": "microbot5-85b6bcc585-zws9j",
    "namespace": "default",
    "operation": "CREATE",
    "userInfo": {
      "username": "system:serviceaccount:kube-system:replicaset-controller",
      "uid": "ce7d5264-51d2-4998-a1db-9d7cd751d167",
      "groups": [
        "system:serviceaccounts",
        "system:serviceaccounts:kube-system",
        "system:authenticated"
      ]
    },
    "object": {
      "kind": "Pod",
      "apiVersion": "v1",
      "metadata": {
        "name": "microbot5-85b6bcc585-zws9j",
        "generateName": "microbot5-85b6bcc585-",
        "namespace": "default",
        "uid": "bcc03889-33be-4390-b047-01d13cf4f51e",
        "creationTimestamp": "2019-10-13T12:14:13Z",
        "labels": {
          "app": "microbot5",
          "pod-template-hash": "85b6bcc585"
        },
        "ownerReferences": [
          {
            "apiVersion": "apps/v1",
            "kind": "ReplicaSet",
            "name": "microbot5-85b6bcc585",
            "uid": "1acfcf3c-2fee-4b31-a4f0-480f4d363ea8",
            "controller": true,
            "blockOwnerDeletion": true
          }
        ]
      },
      "spec": {
        "volumes": [
          {
            "name": "default-token-th7qf",
            "secret": {
              "secretName": "default-token-th7qf"
            }
          }
        ],
        "containers": [
          {
            "name": "microbot",
            "image": "dontrebootme/microbot:v1",
            "resources": {},
            "volumeMounts": [
              {
                "name": "default-token-th7qf",
                "readOnly": true,
                "mountPath": "/var/run/secrets/kubernetes.io/serviceaccount"
              }
            ],
            "terminationMessagePath": "/dev/termination-log",
            "terminationMessagePolicy": "File",
            "imagePullPolicy": "IfNotPresent"
          }
        ],
        "restartPolicy": "Always",
        "terminationGracePeriodSeconds": 30,
        "dnsPolicy": "ClusterFirst",
        "serviceAccountName": "default",
        "serviceAccount": "default",
        "securityContext": {},
        "schedulerName": "default-scheduler",
        "tolerations": [
          {
            "key": "node.kubernetes.io/not-ready",
            "operator": "Exists",
            "effect": "NoExecute",
            "tolerationSeconds": 300
          },
          {
            "key": "node.kubernetes.io/unreachable",
            "operator": "Exists",
            "effect": "NoExecute",
            "tolerationSeconds": 300
          }
        ],
        "priority": 0,
        "enableServiceLinks": true
      },
      "status": {
        "phase": "Pending",
        "qosClass": "BestEffort"
      }
    },
    "oldObject": null,
    "dryRun": false,
    "options": {
      "kind": "CreateOptions",
      "apiVersion": "meta.k8s.io/v1"
    }
  }
}
'


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