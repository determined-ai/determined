.. _controller-reviews:

#############################
 Gateway API Implementations
#############################

This document is a survey of the Gateway API controllers that are available and listed by the `SIG
here <https://gateway-api.sigs.k8s.io/implementations/#haproxy-kubernetes-ingress-controller>`_.

Based on the documentation provided by the projects, we've categorized the implementations into
three groups:

-  **Supported**: The project has implemented the TCPRoute resource and we have tested it.
-  **Support Not Tested**: The project has indicated implementation of the TCPRoute resource but we
   have not tested it.
-  **Not Yet Supported**: The project either has not implemented the TCPRoute resource or has not
   indicated support for it, or we have not found the documentation on it.

*********
 Contour
*********

> Contour v1.29.0 implements Gateway API v1.0.0. All Standard channel v1 API group resources
(GatewayClass, Gateway, HTTPRoute, ReferenceGrant), plus most v1alpha2 API group resources
(TLSRoute, TCPRoute, GRPCRoute, ReferenceGrant, and BackendTLSPolicy) are supported.

`Contour Gateway API Guide <https://projectcontour.io/docs/1.29/guides/gateway-api/>`_

***************
 Envoy Gateway
***************

`Envoy Gateway TCP Routing <https://gateway.envoyproxy.io/latest/tasks/traffic/tcp-routing/>`_

####################
 Support Not Tested
####################

********
 Cilium
********

`Cilium Gateway API
<https://docs.cilium.io/en/stable/network/servicemesh/gateway-api/gateway-api/#gs-gateway-api>`_
Based on Envoy.

********************************
 HAProxy K8s Ingress Controller
********************************

`HAProxy Kubernetes TCPRoute
<https://www.haproxy.com/documentation/kubernetes-ingress/gateway-api/tcproute/>`_ HAProxy
Enterprise Kubernetes Ingress Controller.

******************
 Hashicorp Consul
******************

`Consul TCPRoute Reference
<https://developer.hashicorp.com/consul/docs/k8s/multiport/reference/tcproute>`_

*********
 Traefik
*********

`Traefik Kubernetes Gateway <https://doc.traefik.io/traefik/routing/providers/kubernetes-gateway/>`_
`Traefik Gateway Provider <https://doc.traefik.io/traefik/providers/kubernetes-gateway/>`_ >
Enabling The Experimental Kubernetes Gateway Provider > Since this provider is still experimental,
it needs to be activated in the experimental section of the static configuration.

*******************************************
 Kong Operator and Kong Ingress Controller
*******************************************

`Kong Gateway API <https://docs.konghq.com/gateway-operator/latest/concepts/gateway-api/#main>`_

******
 Kuma
******

Based on Envoy. `Kuma Mesh TCPRoute
<https://kuma.io/docs/2.7.x/policies/meshtcproute/#meshtcproute>`_

*********
 Flomesh
*********

`Flomesh Gateway API Compatibility
<https://github.com/flomesh-io/fsm/blob/main/docs/gateway-api-compatibility.md>`_ Partial tcproute
support

*******
 Istio
*******

`Istio Gateway API Differences
<https://istio.io/latest/docs/tasks/traffic-management/ingress/gateway-api/#differences-from-istio-apis>`_

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

`Apisix Ingress Controller <https://apisix.apache.org/docs/ingress-controller/getting-started/>`_
Mainly ingress focused.

*******
 Azure
*******

`Azure Application Gateway
<https://learn.microsoft.com/en-us/azure/application-gateway/for-containers/overview>`_ No TCPRoute
support.

************
 VMWare Avi
************

Advertises level 4 load balancing but no TCPRoute support yet. Supports k8v1. `VMWare Avi Kubernetes
Guide
<https://docs.vmware.com/en/VMware-Avi-Load-Balancer/1.12/Avi-Kubernetes-Operator-Guide/GUID-84BD68AB-B96F-425C-8323-3A249D6AC8B2.html>`_

***********
 Easegress
***********

No TCPRoute support.

*******************************
 Emissary Ingress - Ambassador
*******************************

No TCPRoute support. `Ambassador Gateway API
<https://www.getambassador.io/docs/edge-stack/latest/topics/using/gateway-api#gateway-api>`_

***********
 Gloo Solo
***********

Based on Envoy but no TCPRoute support.

*****************
 HAProxy Ingress
*****************

No TCPRoute support. `HAProxy Ingress Gateway API
<https://haproxy-ingress.github.io/docs/configuration/gateway-api/>`_

*********
 Linkerd
*********

No TCPRoute support. `Linkerd HTTPRoute Reference <https://linkerd.io/2.15/reference/httproute/>`_

***********
 Litespeed
***********

No TCPRoute support. `Litespeed Kubernetes Gateway
<https://docs.litespeedtech.com/cloud/kubernetes/gateway/>`_

*****************
 Nginx GW Fabric
*****************

No TCPRoute support yet. `Nginx Gateway API Compatibility
<https://docs.nginx.com/nginx-gateway-fabric/overview/gateway-api-compatibility/>`_

*******
 Ngrok
*******

No TCPRoute support. Only HTTRoutes are stable, the others are in an experimental channel. ngrok
supports edges for HTTP/S, TLS, and TCP. The ngrok Operator currently only supports the HTTPRoute.
TLSRoute and TCPRoute will be added after they become stable. `Ngrok Kubernetes Gateway API
<https://ngrok.com/docs/k8s/?k8s-install=gatewayAPI>`_

**********
 WSO2 APK
**********

No TCPRoute support. `WSO2 Kubernetes CRDs
<https://apk.docs.wso2.com/en/latest/catalogs/kubernetes-crds/>`_
