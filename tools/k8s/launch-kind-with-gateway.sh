#!/bin/bash

set -e

if [ -z "$1" ]; then
    echo "Usage: $0 <cluster_name>"
    exit 1
fi

cluster_name=$1
kind create cluster --config - <<EOF
# basic two node config for https://github.com/kubernetes-sigs/kind
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: main # has a mandatory kind prefix already

nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
    protocol: TCP
  - containerPort: 443
    hostPort: 443
    protocol: TCP
- role: worker
  extraPortMappings:
  - containerPort: 22335
    hostPort: 22335
    listenAddress: "0.0.0.0"  

networking:
  disableDefaultCNI: false # use the kindnet CNI
  podSubnet: "10.244.0.0/16" # this is the default flannel subnet
EOF

context_name=kind-$cluster_name

kubectl config use-context $context_name

kubectl apply -f https://projectcontour.io/quickstart/contour-gateway-provisioner.yaml

kubectl apply -f - <<EOF
kind: GatewayClass
apiVersion: gateway.networking.k8s.io/v1
metadata:
  name: contour
spec:
  controllerName: projectcontour.io/gateway-controller
---
kind: Gateway
apiVersion: gateway.networking.k8s.io/v1
metadata:
  name: contour
  namespace: projectcontour
spec:
  gatewayClassName: contour
  listeners:
    - name: tcp
      protocol: TCP
      port: 22335 # Need at least one listener on a gateway. Master will add and patch to it.
      allowedRoutes:
        namespaces:
          from: All
EOF

echo "somehow make the gateway available to the host"
read -p "Press [Enter] once the tunnel has started..."

# somehow make the gateway available to the host
for ((i = 0; i < 60; i++)); do
    export GATEWAY=$(kubectl -n projectcontour get svc envoy-contour)
    if [ -n "$GATEWAY" ]; then
        echo "$GATEWAY"
        break
    fi

    sleep 1
done
