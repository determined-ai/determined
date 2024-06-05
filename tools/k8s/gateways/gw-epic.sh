#!/bin/bash

# https://www.epic-gateway.org/install_k8s_controller/#prerequisites

set -ex

if [ -z "$1" ]; then
    echo "Usage: $0 <minikube_profile>"
    exit 1
fi

K8S_VERSION=${K8S_VERSION:-1.29.5} # https://endoflife.date/kubernetes
minikube_profile=$1
# minikube start --profile $minikube_profile --kubernetes-version $K8S_VERSION

kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.1.0/experimental-install.yaml

kubectl apply -f https://github.com/epic-gateway/puregw/releases/download/v0.27.0/pure-gateway.yaml

# namespace=epic-gateway
namespace=puregw-system

kubectl apply -f - <<EOF
apiVersion: puregw.epic-gateway.org/v1
kind: GatewayClassConfig
metadata:
  name: gatewayhttp
  namespace: $namespace
spec:
  epic:
    user-namespace: root
    service-account: user1
    service-key: yourservicekey

    gateway-hostname: uswest.epick8sgw.io
    gateway-template: gatewayhttp
    cluster-name: mycluster
  trueIngress:
    decapAttachment:
      direction: ingress
      interface: default
      flags: 0
      qid: 0
    encapAttachment:
      direction: egress
      interface: default
      flags: 16
      qid: 1
EOF

kubectl apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: gatewayhttp
spec:
  controllerName: acnodal.io/epic
  # controllerName: epic-gateway.org/epic
  parametersRef:
    name: gatewayhttp
    namespace: $namespace
    group: puregw.epic-gateway.org
    kind: GatewayClassConfig
EOF

kubectl apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: sample-gateway
  namespace: default
spec:
  gatewayClassName: gatewayhttp
  listeners:
    - name: tcp
      protocol: TCP
      port: 52335 # Need at least one listener on a gateway. Master will add and patch to it.
      allowedRoutes:
        namespaces:
          from: All
EOF

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
    export GATEWAY_IP=$(kubectl -n projectcontour get svc envoy-contour -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')
    if [ -n "$GATEWAY_IP" ]; then
        echo "External IP address of envoy-contour service: $GATEWAY_IP"
        break
    fi

    sleep 1
done
