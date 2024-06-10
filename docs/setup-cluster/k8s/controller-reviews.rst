.. _controller-reviews:

#############################
 Gateway API Implementations 
#############################

This document is a survey of the Gateway API controllers that are available and listed by the `SIG
here <https://gateway-api.sigs.k8s.io/implementations/#haproxy-kubernetes-ingress-controller>`_

*********
 Contour
*********

> Contour v1.29.0 implements Gateway API v1.0.0. All Standard channel v1 API group resources
(GatewayClass, Gateway, HTTPRoute, ReferenceGrant), plus most v1alpha2 API group resources
(TLSRoute, TCPRoute, GRPCRoute, ReferenceGrant, and BackendTLSPolicy) are supported.

https://projectcontour.io/docs/1.29/guides/gateway-api/

***************
 Envoy Gateway
***************

https://gateway.envoyproxy.io/latest/tasks/traffic/tcp-routing/

####################
 Support Not Tested
####################

********
 Cilium
********

https://docs.cilium.io/en/stable/network/servicemesh/gateway-api/gateway-api/#gs-gateway-api Based
on Envoy.

********************************
 HAProxy K8s Ingress Controller
********************************

https://www.haproxy.com/documentation/kubernetes-ingress/gateway-api/tcproute/ HAProxy Enterprise
Kubernetes Ingress Controller enterprise only?

******************
 Hashicorp Consul
******************

https://developer.hashicorp.com/consul/docs/k8s/multiport/reference/tcproute

*********
 Traefik
*********

https://doc.traefik.io/traefik/routing/providers/kubernetes-gateway/
https://doc.traefik.io/traefik/providers/kubernetes-gateway/ > Enabling The Experimental Kubernetes
Gateway Provider > Since this provider is still experimental, it needs to be activated in the
experimental section of the static configuration.

*******************************************
 Kong Operator and Kong Ingress Controller
*******************************************

https://docs.konghq.com/gateway-operator/latest/concepts/gateway-api/#main

******
 Kuma
******

Based on Envoy. https://kuma.io/docs/2.7.x/policies/meshtcproute/#meshtcproute

*********
 Flomesh
*********

https://github.com/flomesh-io/fsm/blob/main/docs/gateway-api-compatibility.md partial tcproute
support

*******
 Istio
*******

https://istio.io/latest/docs/tasks/traffic-management/ingress/gateway-api/#differences-from-istio-apis

###################
 Not Yet Supported
###################

**************
 Acnodal Epic
**************

Supports k8s v0.5

***************
 Apache Apisix
***************

https://apisix.apache.org/docs/ingress-controller/getting-started/ mentions
gateway.networking.k8s.io/v1alpha2 mainly ingress focused

*******
 Azure
*******

https://learn.microsoft.com/en-us/azure/application-gateway/for-containers/overview No TCPRoute
support.

************
 VMWare Avi
************

Advertises level 4 load balancing but no TCPRoute support yet. Supports k8v1.
https://docs.vmware.com/en/VMware-Avi-Load-Balancer/1.12/Avi-Kubernetes-Operator-Guide/GUID-84BD68AB-B96F-425C-8323-3A249D6AC8B2.html

***********
 Easegress
***********

No TCPRoute support.

*******************************
 Emissary Ingress - Ambassador
*******************************

No TCPRoute support.
https://www.getambassador.io/docs/edge-stack/latest/topics/using/gateway-api#gateway-api

***********
 Gloo Solo
***********

Based on Envoy but no TCPRoute support.

*****************
 HAProxy Ingress
*****************

No TCPRoute support. https://haproxy-ingress.github.io/docs/configuration/gateway-api/

*********
 Linkerd
*********

No TCPRoute support. Between pods: https://linkerd.io/2.15/features/automatic-mtls/
https://linkerd.io/2.15/reference/httproute/

***********
 Litespeed
***********

https://docs.litespeedtech.com/cloud/kubernetes/gateway/ No TCPRoute support.

*****************
 Nginx GW Fabric
*****************

No TCPRoute support yet.
https://docs.nginx.com/nginx-gateway-fabric/overview/gateway-api-compatibility/

*******
 Ngrok
*******

No TCPRoute support. Only HTTRoutes are stable, the others are in an experimental channel. ngrok
supports edges for HTTP/S, TLS, and TCP. The ngrok Operator currently only supports the HTTPRoute.
TLSRoute and TCPRoute will be added after they become stable.
https://ngrok.com/docs/k8s/?k8s-install=gatewayAPI

**********
 WSO2 APK
**********

No TCPRoute support. https://apk.docs.wso2.com/en/latest/catalogs/kubernetes-crds/
