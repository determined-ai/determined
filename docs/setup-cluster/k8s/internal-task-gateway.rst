Internal Task Gateway
======================

K8s Gateway APIs allow us to expose otherwise internal Determined jobs running on remote K8s clusters
to Determined master and proxies. This is useful for multi-resource manager setups.

The overall setup includes setting up a Gateway controller in the K8s cluster external to Determined
and configuring the Determined master to use the Gateway controller. Please refer to the sections
below to see the configuration changes and controller requirements.


Controller Support - Requirements
=================================

High-level requirements from the controller implementations are:
- Supports Gateway APIs > v1
- Supports TCPRoute and level 4 routing support

TODO inline: supported controllers list from the attached markdown file. `./controller-reviews.md`

In these docs, we'll be using Contour from Project Contour.

Sample Setup - Development
==========================

For internal testing and development, we provide a simple setup script that uses a dynamic
provisioner provided by Project Contour.

On a local dev machine, you can use Minikube and Contour as the controller. We provide a script
to simplify the process. This can be found in `tools/k8s/launch-minikube-with-gateway.sh`.

After you have a working K8s cluster and a Gateway controller running, configure the resource manager
via master config and start the Determined cluster.

Configuration
=============

Below you'll find details on how to configure your cluster and Determined master to use the Internal Task Gateway.

- Total active proxies will be limited by: maxItems set in the Gateway CRD and the portRange configured
  for Determined (not exhaustive).

Master Configuration
---------------------

To configure the optional InternalTaskGateway for a K8s resource manager, you need to add
a struct under `internal_task_gateway` key under each of the desired resource manager configurations.

This is represented by Go package `config.InternalTaskGatewayConfig` defined in `master/internal/config/resource_manager_config.go`

.. code:: go

    // InternalTaskGatewayConfig is config for exposing Determined tasks to outside of the cluster.
    // Useful for multirm when we can only be running in a single cluster.
    type InternalTaskGatewayConfig struct {
        // GatewayName as defined in the k8s cluster.
        GatewayName string `json:"gateway_name"`
        // GatewayNamespace as defined in the k8s cluster.
        GatewayNamespace string `json:"gateway_namespace"`
        GatewayIP        string `json:"gateway_ip"`
        // GWPortStart denotes the inclusive start of the available and exclusive port range to
        // MLDE for InternalTaskGateway.
        GWPortStart int `json:"gateway_port_range_start"`
        // GWPortEnd denotes the inclusive end of the available and exclusive port range to
        // MLDE for InternalTaskGateway.
        GWPortEnd int `json:"gateway_port_range_end"`
    }

Note that the valid port range starts from 1025 to 65535, inclusive.
- CHECK: might want to set max aux containers < min(this and port range)

Gateway
-------

In the CRD `gateways.gateway.networking.k8s.io`
`schema.openAPIV3Schema.properties.spec.properties.listeners.maxItems` defines a max limit of how many
listeners can be active on a single gateway. This limit sets the upper bound on how many tasks can be actively proxied.

Note that when configuring this number you might hit K8s validation complexity thresholds checks. This can
be configured and is dependent on each K8s cluster's requirements and setup.

Other Dev Docs
==============

If you're running Determined outside of the K8s cluster, for example on your local machine for testing and development,
it's possible to test this feature using just a single K8s cluster. All that is needed is for Det master to be sitting external
to the target cluster.

Release Notes
=============

TBD
- Mention docs
- Mention current limitations?


TODO
====
- Update setup guide on multirm 
- Update k8s architecture docs? to include we will deploy services/routes
- Update Carolina’s bug bash docs to include notebook testing
- Helm install path?

