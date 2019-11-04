# Dashboard
Simple server that queries the list of artifacts from a grafain backend and renders a list page.

## How to run it
* Environment

Point `GRAFAINCLI_TM_ADDR` to an grafain backend (minikube example)
```sh
export GRAFAINCLI_TM_ADDR=$(minikube service --url grafain-rpc)
```

* Local

From the project directory
```sh
go run ./cmd/dashboard/
```

* Docker 
```sh

docker run --rm -p8081:8081 alpetest/grafain-dashboard:manual -tm-address=${GRAFAINCLI_TM_ADDR}
```