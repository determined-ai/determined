.. _setup-aks-cluster:

#############################################################
 Set up and Manage an Azure Kubernetes Service (AKS) Cluster
#############################################################

Determined can be installed on a cluster that is hosted on a managed Kubernetes service such as `AKS
<https://azure.microsoft.com/en-us/services/kubernetes-service/>`_. This document describes how to
set up an AKS cluster with GPU-enabled nodes. The recommended setup includes deploying a cluster
with a single non-GPU node that will host the Determined master and database, and an autoscaling
group of GPU nodes. After creating a suitable AKS cluster, you can then proceed with the standard
:ref:`instructions for installing Determined on Kubernetes <install-on-kubernetes>`.

Determined requires GPU-enabled nodes and the Kubernetes cluster to be running version >= 1.19 and
<= 1.21, though later versions may work.

***************
 Prerequisites
***************

To deploy an AKS cluster, the user must have a resource group to manage the resources consumed by
the cluster. To create one, follow the instructions found in the `Azure Resource Groups
Documentation
<https://docs.microsoft.com/en-us/azure/azure-resource-manager/management/manage-resource-groups-portal#create-resource-groups>`_.

Additionally, users must have the `Azure CLI
<https://docs.microsoft.com/en-us/cli/azure/install-azure-cli>`_ and `kubectl
<https://kubernetes.io/docs/tasks/tools/install-kubectl/>`_ installed on their local machine.

Finally, authenticate with the Azure CLI using ``az login`` in order to have access to your Azure
subscription.

********************
 Set up the Cluster
********************

.. code:: bash

   # Specify the Azure Resource Group you will be using to deploy the cluster.
   AKS_RESOURCE_GROUP=<resource group name, e.g. "determined-resource-group">

   # Set a unique name for your cluster.
   AKS_CLUSTER_NAME=<any unique name, e.g. "determined-cluster">

   # Set a unique name for your node pool. Azure requires node pool names to consist
   # solely of alphanumeric characters, start with a lowercase letter, and
   # be no longer than 12 characters.
   AKS_GPU_NODE_POOL_NAME=<any unique, conforming, name, e.g. "determined-node-pool">

   # Set the GPU VM Size for your node pool. This VM size corresponds to a machine with 4 Tesla K80 GPUs.
   GPU_VM_SIZE=Standard_NC24

   # Launch the AKS cluster that will contain a single non-GPU node.
   az aks create --resource-group ${AKS_RESOURCE_GROUP} --name ${AKS_CLUSTER_NAME} \
    --node-count 1 --generate-ssh-keys --vm-set-type VirtualMachineScaleSets \
    --load-balancer-sku standard --node-vm-size Standard_D8_v3

   # Create a node pool. This will not launch any nodes immediately but will
   # scale up and down as needed. If you change the GPU type or the number of
   # GPUs per node, you may need to change the machine-type.
   az aks nodepool add --resource-group ${AKS_RESOURCE_GROUP} --cluster-name ${AKS_CLUSTER_NAME} \
    --name ${AKS_GPU_NODE_POOL_NAME} --node-count 0 --node-vm-size ${GPU_VM_SIZE} \
    --enable-cluster-autoscaler --min-count 0 --max-count 4

*****************************
 Create a kubeconfig for AKS
*****************************

After creating the cluster, ``kubectl`` should be used to deploy apps. In order for ``kubectl`` to
be used with AKS, users need to create or update the cluster kubeconfig. This can be done with the
command:

.. code:: bash

   az aks get-credentials --resource-group ${AKS_RESOURCE_GROUP} --name ${AKS_CLUSTER_NAME}

********************
 Enable GPU Support
********************

To allow the AKS cluster to recognize GPU hardware resources, refer to the instructions provided by
Azure on the `Install NVIDIA Device Plugin
<https://docs.microsoft.com/en-us/azure/aks/gpu-cluster#install-nvidia-device-plugin>`_ tutorial.

With this, the cluster is fully set up, and Determined can be deployed onto it.

***********************
 Manage an AKS Cluster
***********************

Update the Autoscaler
=====================

To update the cluster autoscaler, use the following Azure CLI command:

.. code:: bash

   az aks nodepool update --update-cluster-autoscaler --min-count <new_min_count> \
   --max-count <new_max_count> --resource-group ${AKS_RESOURCE_GROUP} \
   --cluster-name ${AKS_CLUSTER_NAME} --name ${AKS_GPU_NODE_POOL_NAME}

Add Taints and Tolerations to Nodes
===================================

For general instructions on adding taints and tolerations to nodes, see the :ref:`Taints and
Tolerations <taints-on-kubernetes>` section in our :ref:`Guide to Kubernetes
<install-on-kubernetes>`. There, you can find an explanation of taints and tolerations, as well as
instructions for using ``kubectl`` to add them to existing clusters.

It is important to note that if you use the Azure CLI to create nodes with taints, you must also add
tolerations using ``kubectl``; otherwise, Kubernetes will be unable to schedule pods on the tainted
node.

To create a nodepool with a taint in AKS, use the ``--node-taints`` flag to specify the type, tag,
and effect:

.. code:: bash

   az aks nodepool add \
      --resource-group ${AKS_RESOURCE_GROUP} \
      --cluster-name ${AKS_CLUSTER_NAME} \
      --name ${AKS_NODE_POOL_NAME} \
      --node-count 1 \
      --node-taints ${TAINT_TYPE}=${TAINT_TAG}:{TAINT_EFFECT} \
      --no-wait

The following CLI command is an example of using the ``az`` CLI to make a node that is unschedulable
unless a Pod has a toleration for a taint with type ``sku`` equal to ``gpu`` with the ``NoSchedule``
effect.

.. code:: bash

   az aks nodepool add \
   --resource-group ${AKS_RESOURCE_GROUP} \
   --cluster-name ${AKS_CLUSTER_NAME} \
   --name ${AKS_NODE_POOL_NAME} \
   --node-count 1 \
   --node-taints sku=gpu:NoSchedule \
   --no-wait

Delete the Cluster
==================

To delete the AKS cluster, use the following Azure CLI command:

.. code:: bash

   az aks delete --resource-group ${AKS_RESOURCE_GROUP} --name ${AKS_CLUSTER_NAME}

************
 Next Steps
************

-  :ref:`install-on-kubernetes`
-  :ref:`k8s-dev-guide`
