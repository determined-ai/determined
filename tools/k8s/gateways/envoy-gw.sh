#!/bin/bash

# https://www.epic-gateway.org/install_k8s_controller/#prerequisites

set -ex

if [ -z "$1" ]; then
    echo "Usage: $0 <minikube_profile>"
    exit 1
fi

K8S_VERSION=${K8S_VERSION:-1.29.5} # https://endoflife.date/kubernetes
minikube_profile=$1
minikube start --profile $minikube_profile --kubernetes-version $K8S_VERSION

# minikube addons enable metallb
# # see what changes would be made, returns nonzero returncode if different
# kubectl get configmap kube-proxy -n kube-system -o yaml | \
# sed -e "s/strictARP: false/strictARP: true/" | \
# kubectl diff -f - -n kube-system

# # actually apply the changes, returns nonzero returncode on errors only
# kubectl get configmap kube-proxy -n kube-system -o yaml | \
# sed -e "s/strictARP: false/strictARP: true/" | \
# kubectl apply -f - -n kube-system

# https://gateway.envoyproxy.io/latest/install/gateway-helm-api/
helm install eg oci://docker.io/envoyproxy/gateway-helm --version v0.0.0-latest -n envoy-gateway-system --create-namespace
# kubectl apply --server-side -f https://github.com/envoyproxy/gateway/releases/download/latest/install.yaml

kubectl wait --timeout=5m -n envoy-gateway-system deployment/envoy-gateway --for=condition=Available

kubectl apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: eg
spec:
  controllerName: gateway.envoyproxy.io/gatewayclass-controller
---
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: eg
spec:
  gatewayClassName: eg
  listeners:
    - name: http
      protocol: HTTP
      port: 80
EOF

# envoy service name
export ENVOY_SERVICE=$(kubectl get svc -n envoy-gateway-system --selector=gateway.envoyproxy.io/owning-gateway-namespace=default,gateway.envoyproxy.io/owning-gateway-name=eg -o jsonpath='{.items[0].metadata.name}')

if sudo -n true 2>/dev/null; then
    # Either like have a smaller subnet so we don't conflict. Or like don't start it for the second one.
    nohup minikube --profile $minikube_profile tunnel & # TODO won't work for users with sudo passwords.
else
    echo "sudo password is required to start the tunnel."
    echo "Please run the following command separately to start the tunnel:"
    echo "minikube --profile $minikube_profile tunnel"
    read -p "Press [Enter] once the tunnel has started..."
fi

for ((i = 0; i < 60; i++)); do
    export GATEWAY_IP=$(kubectl -n envoy-gateway-system get svc $ENVOY_SERVICE -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')
    if [ -n "$GATEWAY_IP" ]; then
        echo "External IP address of envoy-contour service: $GATEWAY_IP"
        break
    fi

    sleep 1
done
