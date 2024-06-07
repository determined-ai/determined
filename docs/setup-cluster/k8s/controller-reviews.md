# Gateway controller survey

https://gateway-api.sigs.k8s.io/implementations/#haproxy-kubernetes-ingress-controller

looking for: level 4, supports tcproute, supports https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.0.0 and newer

## [y] contour
Contour v1.29.0 implements Gateway API v1.0.0. All Standard channel v1 API group resources (GatewayClass, Gateway, HTTPRoute, ReferenceGrant), plus most v1alpha2 API group resources (TLSRoute, TCPRoute, GRPCRoute, ReferenceGrant, and BackendTLSPolicy) are supported.
https://projectcontour.io/docs/1.29/guides/gateway-api/

## [y] envoy gateway
https://gateway.envoyproxy.io/latest/tasks/traffic/tcp-routing/
tcproute support

## [y] cilium
they support tcproute
https://docs.cilium.io/en/stable/network/servicemesh/gateway-api/gateway-api/#gs-gateway-api
uses envoy

## [y] haproxy k8s ingress controller
https://www.haproxy.com/documentation/kubernetes-ingress/gateway-api/tcproute/
HAProxy Enterprise Kubernetes Ingress Controller
enterprise only?

## [y?$] hashicorp consul
TCPRoute but some options are gated?
https://developer.hashicorp.com/consul/docs/k8s/multiport/reference/tcproute

## [y] traefik
has tcproute support
https://doc.traefik.io/traefik/routing/providers/kubernetes-gateway/
https://doc.traefik.io/traefik/providers/kubernetes-gateway/ 
> Enabling The Experimental Kubernetes Gateway Provider
> Since this provider is still experimental, it needs to be activated in the experimental section of the static configuration.

## [y] kong operator and kong ingress controller
has tcproute support
https://docs.konghq.com/gateway-operator/latest/concepts/gateway-api/#main

## [y] kuma
envoy based. has tcproute? https://kuma.io/docs/2.7.x/policies/meshtcproute/#meshtcproute

## [y] flomesh
https://github.com/flomesh-io/fsm/blob/main/docs/gateway-api-compatibility.md
partial tcproute support

## [?] istio
https://istio.io/latest/docs/tasks/traffic-management/ingress/gateway-api/#differences-from-istio-apis

# Not Yet Supported

## acnodal epic
supports k8s v0.5 

## apache apisix
https://apisix.apache.org/docs/ingress-controller/getting-started/ mentions 
gateway.networking.k8s.io/v1alpha2

mainly ingress focused

## azure
https://learn.microsoft.com/en-us/azure/application-gateway/for-containers/overview
no tcproute

## wmware avi
advertises level 4 load balancing but
no tpcroute support. supports k8v1


https://docs.vmware.com/en/VMware-Avi-Load-Balancer/1.12/Avi-Kubernetes-Operator-Guide/GUID-84BD68AB-B96F-425C-8323-3A249D6AC8B2.html


## Easegress
no tcproute

## emissary ingress - ambassador

no tcproute
https://www.getambassador.io/docs/edge-stack/latest/topics/using/gateway-api#gateway-api


## Gloo solo
uses envoy
no tcproute


## haproxy ingress
no tcproute
https://haproxy-ingress.github.io/docs/configuration/gateway-api/

## linkerd
no tcproute
between pods https://linkerd.io/2.15/features/automatic-mtls/
https://linkerd.io/2.15/reference/httproute/
## litespeed
https://docs.litespeedtech.com/cloud/kubernetes/gateway/
no tcproute

## nginx gw fabric 
no tcproute yet
https://docs.nginx.com/nginx-gateway-fabric/overview/gateway-api-compatibility/
https://ngrok.com/docs/k8s/with-edges/#gateway-api

## ngrok
no tcproute
Only HTTRoutes are stable, the others are in an experimental channel. ngrok supports edges for HTTP/S, TLS, and TCP. The ngrok Operator currently only supports the HTTPRoute. TLSRoute and TCPRoute will be added after they become stable.
https://ngrok.com/docs/k8s/?k8s-install=gatewayAPI

## stuner
no

## wso2 apk
no tcproute
https://apk.docs.wso2.com/en/latest/catalogs/kubernetes-crds/
