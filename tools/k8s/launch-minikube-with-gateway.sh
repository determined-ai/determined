#!/bin/bash
if [ -z "$1" ]; then
    echo "Usage: $0 <minikube_profile>"
    exit 1
fi

minikube_profile=$1
minikube start --profile $minikube_profile

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

# Either like have a smaller subnet so we don't conflict. Or like don't start it for the second one.
nohup minikube --profile $minikube_profile tunnel & # TODO won't work for users with sudo passwords.

for ((i = 0; i < 60; i++)); do
    export GATEWAY_IP=$(kubectl -n projectcontour get svc envoy-contour -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')
    if [ -n "$GATEWAY_IP" ]; then
        echo "External IP address of envoy-contour service: $GATEWAY_IP"
        break
    fi

    sleep 1
done
