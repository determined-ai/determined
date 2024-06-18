.. _internal-task-gateway:

#######################
 Internal Task Gateway
#######################

`K8s Gateway APIs <https://gateway-api.sigs.k8s.io/>`_ allow us to expose otherwise internal
Determined jobs running on remote K8s clusters to Determined master and proxies. This is useful for
multi-resource manager setups.

The overall setup includes installing and configuring a Gateway controller in the K8s cluster
external to Determined and configuring the Determined master to use the Gateway controller. Please
refer to the sections below to see the configuration changes and controller requirements.

.. warning::

   This feature exposes Determined tasks to the outside world. Please ensure that you have the
   necessary security measures in place to limit access to the exposed tasks. This could include
   setting up a firewall, using a VPN, IP whitelisting, or other security measures.

Limitations:

-  Exposing proxies in multi-slot jobs is not supported. Currently this only includes experiments
   running distributed training that want to manually expose proxies to the outside world.

###################################
 Controller Support - Requirements
###################################

High-level requirements from the controller implementations are:

-  Supports Gateway APIs > v1

-  Supports `TCPRoute
   <https://gateway-api.sigs.k8s.io/concepts/api-overview/#tcproute-and-udproute>`_ and level 4
   routing support

Take a look at our current survey of the existing implementations here: :doc:`controller-reviews`

In these docs, we'll be using `Contour from Project Contour <https://projectcontour.io/>`_.

############################
 Sample Setup - Development
############################

For internal testing and development, we provide a simple setup script that uses a dynamic
provisioner provided by Project Contour.

On a local dev machine, you can use Minikube and Contour as the controller. We provide a script to
simplify the process. This can be found in `tools/k8s/launch-minikube-with-gateway.sh`.

After you have a working K8s cluster and a Gateway controller running, configure the resource
manager via master config and start the Determined cluster.

###############
 Configuration
###############

Below you'll find details on how to configure your cluster and Determined master to use the Internal
Task Gateway.

-  Total active proxies will be limited by: maxItems set in the Gateway CRD and the portRange
   configured for Determined (not exhaustive).

**********************
 Master Configuration
**********************

To configure the optional InternalTaskGateway for a K8s resource manager, you need to add a struct
under `internal_task_gateway` key under each of the desired resource manager configurations.

This is represented by Go package `config.InternalTaskGatewayConfig` defined in
`master/internal/config/resource_manager_config.go`

.. code:: yaml

   internal_task_gateway:
     # GatewayName as defined in the k8s cluster.
     gateway_name: <GatewayName>

     # GatewayNamespace as defined in the k8s cluster.
     gateway_namespace: <GatewayNamespace>

     # GatewayIP as defined in the k8s cluster.
     gateway_ip: <GatewayIP>

     # GWPortStart denotes the inclusive start of the available and exclusive port range to
     # MLDE for InternalTaskGateway.
     gateway_port_range_start: <GWPortStart>

     # GWPortEnd denotes the inclusive end of the available and exclusive port range to
     # MLDE for InternalTaskGateway.
     gateway_port_range_end: <GWPortEnd>

-  Valid port range starts from 1025 to 65535, inclusive.
-  GatewayIP is the IP address of the Gateway controller that is visible to the Determined master.

*********
 Gateway
*********

In the CRD `gateways.gateway.networking.k8s.io`
`schema.openAPIV3Schema.properties.spec.properties.listeners.maxItems` defines a max limit of how
many listeners can be active on a single gateway. This limit sets the upper bound on how many tasks
can be actively proxied.

Note that when configuring this number you might hit K8s validation complexity thresholds checks.
This can be configured and is dependent on each K8s cluster's requirements and setup.

For example to up the number from allowed listeners to 128, you can modify the CRD at the given path
above and `kubectl apply -f <path-to-crd>`. Make sure to set the value for the version of the spec
that your Gateway API is going to use.

################
 Other Dev Docs
################

If you're running Determined outside of the K8s cluster, for example on your local machine for
testing and development, it's possible to test this feature using just a single K8s cluster. All
that is needed is for Det master to be sitting external to the target cluster.

###############
 Release Notes
###############

TBD

-  Mention docs
-  Mention current limitations?
