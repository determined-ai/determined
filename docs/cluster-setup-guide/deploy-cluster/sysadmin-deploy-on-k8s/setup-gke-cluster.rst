.. _setup-gke-cluster:

############################################################
 Set up and Manage a Google Kubernetes Engine (GKE) Cluster
############################################################

Determined can be installed on a cluster that is hosted on a managed Kubernetes service such as `GKE
<https://cloud.google.com/kubernetes-engine>`_. This document describes how to set up a GKE cluster
with GPU-enabled nodes. The recommended setup includes deploying a cluster with a single non-GPU
node that will host the Determined master and database, and an autoscaling group of GPU nodes. After
creating a suitable GKE cluster, you can then proceed with the standard :ref:`instructions for
installing Determined on Kubernetes <install-on-kubernetes>`.

Determined requires GPU-enabled nodes and the Kubernetes cluster to be running version >= 1.19 and
<= 1.21, though later versions may work.

***************
 Prerequisites
***************

Before setting up a GKE cluster, the user should have `Google Cloud SDK
<https://cloud.google.com/sdk/docs/quickstarts/>`_ and `kubectl
<https://kubernetes.io/docs/tasks/tools/install-kubectl/>`_ installed on their local machine.

********************
 Set up the Cluster
********************

.. code:: bash

   # Set a unique name for your cluster.
   GKE_CLUSTER_NAME=<any unique name, e.g. "determined-cluster">

   # Set a unique name for your node pool.
   GKE_GPU_NODE_POOL_NAME=<any unique name, e.g., "determined-node-pool">

   # Set a unique name for the GCS bucket that will store your checkpoints.
   # When installing Determined, set checkpointStorage.bucket to the value defined here.
   GCS_BUCKET_NAME=<any unique name, e.g., "determined-checkpoint-bucket">

   # Set the GPU type for your node pool. Other options include p100, p4, and v100.
   GPU_TYPE=nvidia-tesla-t4

   # Set the number of GPUs per node.
   GPUS_PER_NODE=4

   # Launch the GKE cluster that will contain a single non-GPU node.
   gcloud container clusters create ${GKE_CLUSTER_NAME} \
       --region us-west1 \
       --node-locations us-west1-b\
       --num-nodes=1 \
       --machine-type=n1-standard-16

   # Create a node pool. This will not launch any nodes immediately but will
   # scale up and down as needed. If you change the GPU type or the number of
   # GPUs per node, you may need to change the machine-type.
   gcloud container node-pools create ${GKE_GPU_NODE_POOL_NAME} \
     --cluster ${GKE_CLUSTER_NAME} \
     --accelerator type=${GPU_TYPE},count=${GPUS_PER_NODE} \
     --zone us-west1 \
     --num-nodes=0 \
     --enable-autoscaling \
     --min-nodes=0 \
     --max-nodes=4 \
     --machine-type=n1-standard-32 \
     --scopes=storage-full,cloud-platform

   # Deploy a DaemonSet that enables the GPUs.
   kubectl apply -f https://raw.githubusercontent.com/GoogleCloudPlatform/container-engine-accelerators/master/nvidia-driver-installer/cos/daemonset-preloaded.yaml

   # Create a GCS bucket to store checkpoints.
   gsutil mb gs://${GCS_BUCKET_NAME}

**********************
 Manage a GKE Cluster
**********************

For general instructions on adding taints and tolerations to nodes, see the :ref:`Taints and
Tolerations <taints-on-kubernetes>` section in our :ref:`Guide to Kubernetes
<install-on-kubernetes>`. There, you can find an explanation of taints and tolerations, as well as
instructions for using ``kubectl`` to add them to existing clusters.

It is important to note that if you use the ``gcloud`` CLI to create nodes with taints, you must
also add tolerations using ``kubectl``; otherwise, Kubernetes will be unable to schedule pods on the
tainted node.

To create a nodepool or a cluster with a taint in GKE, use the ``--node-taints`` flag to specify the
type, tag, and effect.

.. code:: bash

   gcloud container clusters create ${GKE_CLUSTER_NAME} \
      --node-taints ${TAINT_TYPE}=${TAINT_TAG}:${TAINT_EFFECT}

The following command is an example of using the ``gcloud`` CLI to make a cluster that with a taint
with type ``dedicated`` equal to ``experimental`` with the ``PreferNoSchedule`` effect.

.. code:: bash

   gcloud container clusters create ${GKE_CLUSTER_NAME} \
      --node-taints dedicated=experimental:PreferNoSchedule

.. code:: bash

   gcloud container node-pools create ${GKE_NODE_POOL_NAME} \
      --cluster ${GKE_CLUSTER_NAME} \
      --node-taints ${TAINT_TYPE}=${TAINT_TAG}:${TAINT_EFFECT}

The following CLI command is an example of using the ``gcloud`` CLI to make a node with a taint with
type ``special`` equal to ``gpu`` with the ``NoExecute`` effect.

.. code:: bash

   gcloud container node-pools create ${GKE_NODE_POOL_NAME} \
      --cluster ${GKE_CLUSTER_NAME} \
      --node-taints special=gpu:NoExecute

************
 Next Steps
************

-  :ref:`install-on-kubernetes`
-  :ref:`k8s-dev-guide`
