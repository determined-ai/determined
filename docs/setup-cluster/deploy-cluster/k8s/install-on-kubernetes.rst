.. _install-on-kubernetes:

##################################
 Install Determined on Kubernetes
##################################

+-----------------------------------------------------------------+
| Configuration Reference                                         |
+=================================================================+
| :doc:`/reference/deploy/config/helm-config-reference`           |
+-----------------------------------------------------------------+

This user guide describes how to install Determined on `Kubernetes <https://kubernetes.io/>`__.
using the :download:`Determined Helm Chart </helm/determined-latest.tgz>`.

When the Determined Helm chart is installed, the following entities will be created:

-  Deployment of the Determined master.
-  ConfigMap containing configurations for the Determined master.
-  LoadBalancer service to make the Determined master accessible. Later in this guide, we describe
   how it is possible to replace this with a NodePort service.
-  ServiceAcccount which will be used by the Determined master.
-  Deployment of a Postgres database. Later in this guide, we describe how an external database can
   be used instead.
-  PersistentVolumeClaim for the Postgres database. Omitted if using an external database.
-  Service to allow the Determined master to communicate with the Postgres database. Omitted if
   using an external database.

When installing :ref:`Determined on Kubernetes <install-on-kubernetes>` using Helm, the deployment
should be configured by editing the ``values.yaml`` and ``Chart.yaml`` files in the
:download:`Determined Helm Chart </helm/determined-latest.tgz>`.

***************
 Prerequisites
***************

Before installing Determined on a Kubernetes cluster, please ensure that the following prerequisites
are satisfied:

-  The Kubernetes cluster should be running Kubernetes version >= 1.19 and <= 1.21, though later
   versions may work.

-  You should have access to the cluster via `kubectl
   <https://kubernetes.io/docs/tasks/tools/install-kubectl/>`_.

-  `Helm 3 <https://helm.sh/docs/intro/install/>`_ should be installed.

-  If you are using a private image registry or the enterprise edition, you should add a secret
   using `kubectl create secret
   <https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/>`_.

-  The nodes in the cluster already have or can pull the ``fluent/fluent-bit:1.9.3`` Docker image
   from Docker Hub.

-  Optional: for GPU-based training, the Kubernetes cluster should have `GPU support
   <https://kubernetes.io/docs/tasks/manage-gpus/scheduling-gpus/>`_ enabled.

You should also download a copy of the :download:`Determined Helm Chart
</helm/determined-latest.tgz>` and extract it on your local machine.

If you do not yet have a Kubernetes cluster deployed and you want to use Determined in a public
cloud environment, we recommend using a managed Kubernetes offering such as `Google Kubernetes
Engine (GKE) <https://cloud.google.com/kubernetes-engine>`__ on GCP or `Elastic Kubernetes Service
(EKS) <https://aws.amazon.com/eks/>`__ on AWS. For more info on configuring GKE for use with
Determined, refer to the :ref:`Instructions for setting up a GKE cluster <setup-gke-cluster>`. For
info on configuring EKS, refer to the :ref:`Instructions for setting up an EKS cluster
<setup-eks-cluster>`.

***************
 Configuration
***************

When installing Determined using Helm, first configure some aspects of the Determined deployment by
editing the ``values.yaml`` and ``Chart.yaml`` files in the Helm chart.

Image Registry Configuration
============================

To configure which image registry of Determined will be installed by the Helm chart, change
``imageRegistry`` in ``values.yaml``. You can specify the Docker Hub public registry
``determinedai`` or specify any private registry that hosts the Determined master image.

Image Pull Secret Configuration
===============================

To configure which image pull secret will be used by the Helm chart, change ``imagePullSecretName``
in ``values.yaml``. You can set it to empty for the Docker Hub public registry or specify any secret
that is configured using `kubectl create secret
<https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/>`_.

.. _configure-determined-kubernetes-version:

Version Configuration
=====================

To configure which version of Determined will be installed by the Helm chart, change ``appVersion``
in ``Chart.yaml``. You can specify a release version (e.g., ``0.13.0``) or specify any commit hash
from the `upstream Determined repo <https://github.com/determined-ai/determined>`_ (e.g.,
``b13461ed06f2fad339e179af8028d4575db71a81``). You are strongly encouraged to use a released
version.

Resource Configuration (GPU-based setups)
=========================================

For GPU-based configurations, you must specify the number of GPUs on each node (for GPU-enabled
nodes only). This is done by setting ``maxSlotsPerPod`` in ``values.yaml``. Determined uses this
information when scheduling multi-GPU tasks. Each multi-GPU (distributed training) task will be
scheduled as a set of ``slotsPerTask / maxSlotsPerPod`` separate pods, with each pod assigned up to
``maxSlotsPerPod`` GPUs. Distributed tasks with sizes that are not divisible by ``maxSlotsPerPod``
are never scheduled. If you have a cluster of different size nodes, set ``maxSlotsPerPod`` to the
greatest common divisor of all the sizes. For example, if you have some nodes with 4 GPUs and other
nodes with 8 GPUs, set ``maxSlotsPerPod`` to ``4`` so that all distributed experiments will launch
with 4 GPUs per pod (with two pods on 8-GPU nodes).

Resource Configuration (CPU-based setups)
=========================================

For CPU-only configurations, you need to set ``slotType: cpu`` as well as
``slotResourceRequests.cpu: <number of CPUs per slot>`` in ``values.yaml``. Please note that the
number of CPUs allocatable by Kubernetes may be lower than the number of "hardware" CPU cores. For
example, an 8-core node may provide 7.91 CPUs, with the rest allocated for the Kubernetes system
tasks. If ``slotResourceRequests.cpu`` was set to 8 in this example, the pods would fail to
allocate, so it should be set to a lower number instead, such as 7.5.

Then, similarly to GPU-based configuration, ``maxSlotsPerPod`` needs to be set to the greatest
common divisor of all the node sizes. For example, if you have 16-core nodes with 15 allocatable
CPUs, it's reasonable to set ``maxSlotsPerPod: 1`` and ``slotResourceRequests.cpu: 15``. If you have
some 32-core nodes and some 64-core nodes, and you want to use finer-grained
``slotResourceRequests.cpu: 15``, set ``maxSlotsPerPod: 2``.

Checkpoint Storage
==================

Checkpoints and TensorBoard events can be configured to be stored in ``shared_fs``, `AWS S3
<https://aws.amazon.com/s3/>`__, `Microsoft Azure Blob Storage
<https://azure.microsoft.com/en-us/services/storage/blobs>`__, or `GCS
<https://cloud.google.com/storage>`__. By default, checkpoints and TensorBoard events are stored
using ``shared_fs``, which creates a `hostPath Volume
<https://kubernetes.io/docs/concepts/storage/volumes/#hostpath>`__ and saves to the host file
system. This configuration is intended for *initial testing only*; you are strongly discouraged from
using ``shared_fs`` for actual deployments of Determined on Kubernetes, because most Kubernetes
cluster nodes do not have a shared file system.

Instead of using ``shared_fs``, configure either AWS S3, Microsoft Azure Blob Storage, or GCS:

-  **AWS S3**: To configure Determined to use AWS S3 for checkpoint and TensorBoard storage, you
   need to set ``checkpointStorage.type`` in ``values.yaml`` to ``s3`` and set
   ``checkpointStorage.bucket`` to the name of the bucket. The pods launched by the Determined
   master must have read, write, and delete access to the bucket. To enable this you can optionally
   configure ``checkpointStorage.accessKey`` and ``checkpointStorage.secretKey``. You can optionally
   configure ``checkpointStorage.endpointUrl`` which specifies the endpoint to use for S3 clones
   (e.g., ``http://<minio-endpoint>:<minio-port|default=9000>``).

-  **Microsoft Azure Blob Storage**: To configure Determined to use Microsoft Azure Blob Storage for
   checkpoint and TensorBoard storage, you need to set ``checkpointStorage.type`` in ``values.yaml``
   to ``azure`` and set ``checkpointStorage.container`` to the name of the container to store it in.
   You must also specify one of ``connection_string`` - the connection string associated with the
   Azure Blob Storage service account to use, or the tuple ``account_url`` and ``credential`` -
   where ``account_url`` is the URL for the service account to use, and ``credential`` is an
   optional credential.

-  **GCS**: To configure Determined to use Google Cloud Storage for checkpoints and TensorBoard
   data, set ``checkpointStorage.type`` in ``values.yaml`` to ``gcs`` and set
   ``checkpointStorage.bucket`` to the name of the bucket. The pods launched by the Determined
   master must have read, write, and delete access to the bucket. For example, when launching `GKE
   nodes <https://cloud.google.com/sdk/gcloud/reference/container/node-pools/create>`__ you need to
   specify ``--scopes=storage-full`` to configure proper GCS access.

Default Pod Specs (Optional)
============================

As described in the :ref:`determined-on-kubernetes` guide, when tasks (e.g., experiments, notebooks)
are started in a Determined cluster running on Kubernetes, the Determined master launches pods to
execute these tasks. The Determined helm chart makes it possible to set default pod specs for all
CPU and GPU tasks. The defaults can be defined in ``values.yaml`` under
``taskContainerDefaults.cpuPodSpec`` and ``taskContainerDefaults.gpuPodSpec``. For examples of how
to do this and a description of permissible fields, see the :ref:`specifying custom pod specs
<custom-pod-specs>` guide.

Default Password (Optional)
===========================

Unless otherwise specified, the pre-existing users, ``admin`` and ``determined``, do not have
passwords associated with their accounts. You can set a default password for the ``determined`` and
``admin`` accounts if preferred or needed. This password will not affect any other user account. For
additional information on managing users in determined, see the :ref:`topic guide on users <users>`.

Database (Optional)
===================

By default, the Helm chart deploys an instance of Postgres on the same Kubernetes cluster where
Determined is deployed. If this is not what you want, you can configure the Helm chart to use an
external Postgres database by setting ``db.hostAddress`` to the IP address of their database. If
``db.hostAddress`` is configured, the Determined Helm chart will not deploy a database.

.. _tls-on-kubernetes:

TLS (Optional)
==============

By default, the Helm chart will deploy a load-balancer which makes the Determined master accessible
over HTTP. To secure your cluster, Determined supports configuring `TLS encryption
<https://en.wikipedia.org/wiki/Transport_Layer_Security>`__ which can be configured to terminate
inside a load-balancer or inside the Determined master itself. To configure TLS, set
``useNodePortForMaster`` to ``true``. This will instruct Determined to deploy a NodePort service for
the master. You can then configure an `Ingress
<https://kubernetes.io/docs/concepts/services-networking/ingress/#tls>`__ that performs TLS
termination in the load balancer and forwards plain text to the NodePort service, or forwards TLS
encrypted data. Please note when configuring an Ingress that you need to have an `Ingress controller
<https://github.com/bitnami/charts/tree/master/bitnami/nginx-ingress-controller>`__ running your
cluster.

#. **TLS termination in a load-balancer (e.g., nginx).** This option will provide TLS encryption
   between the client and the load-balancer, with all communication inside the cluster performed via
   HTTP. To configure this option set ``useNodePortForMaster`` to ``true`` and then configure an
   Ingress service to perform TLS termination and forward the plain text traffic to the Determined
   master.

#. **TLS termination in the Determined master.** This option will provide TLS encryption inside the
   Kubernetes cluster. All communication with the master will be encrypted. Communication between
   task containers (distributed training) will not be encrypted. To configure this option create a
   Kubernetes TLS secret within the namespace where Determined is being installed and set
   ``tlsSecret`` to be the name of this secret. You also need to set ``useNodePortForMaster`` to
   ``true``. After the NodePort service is created, you can configure an Ingress to forward TLS
   encrypted data to the NodePort service.

An example of how to configure an Ingress, which will perform TLS termination in the load-balancer
by default:

.. code:: yaml

   apiVersion: networking.k8s.io/v1beta1
   kind: Ingress
   metadata:
     name: determined-ingress
     annotations:
       kubernetes.io/ingress.class: "nginx"

       # Uncommenting this option instructs the created load-balancer
       # to forward TLS encrypted data to the NodePort service and
       # perform TLS termination in the Determined master. In order
       # to configure ssl-passthrough, your nginx ingress controller
       # must be running with the --enable-ssl-passthrough option enabled.
       #
       # nginx.ingress.kubernetes.io/ssl-passthrough: "true"
   spec:
     tls:
     - hosts:
       - your-hostname-for-determined.ai
       secretName: your-tls-secret-name
     rules:
     - host: your-hostname-for-determined.ai
       http:
         paths:
           - path: /
             backend:
               serviceName: determined-master-service-<name for your deployment>
               servicePort: masterPort configured in values.yaml

To see information about using AWS Load Balancer instead of nginx visit :ref:`Using AWS Load
Balancer <aws-lb>`.

Default Scheduler (Optional)
============================

Determined includes support for the `lightweight coscheduling plugin
<https://github.com/kubernetes-sigs/scheduler-plugins/tree/release-1.18/pkg/coscheduling>`__, which
extends the default Kubernetes scheduler to provide gang scheduling. This feature is currently in
beta and is not enabled by default. To activate the plugin, set the ``defaultScheduler`` field to
``coscheduler``. If the field is empty or doesn't exist, Determined will use the default Kubernetes
scheduler to schedule all experiments and tasks.

.. code:: yaml

   defaultScheduler: coscheduler

Determined also includes support for priority-based scheduling with preemption. This feature allows
experiments to be preempted if higher priority ones are submitted. This feature is also in beta and
is not enabled by default. To activate priority-based preemption scheduling, set
``defaultScheduler`` to ``preemption``.

.. code:: yaml

   defaultScheduler: preemption

.. _taints-on-kubernetes:

Node Taints
===========

Tainting nodes is optional, but you might want to taint nodes to restrict which nodes a pod may be
scheduled onto. A taint consists of a taint type, tag, and effect.

When using a managed kubernetes cluster (e.g. a :ref:`GKE <setup-gke-cluster>`, :ref:`AKS
<setup-aks-cluster>`, or :ref:`EKS <setup-eks-cluster>` cluster), it is possible to specify taints
at cluster or nodepool creation using the specified CLIs. Please refer to the set up pages for each
managed cluster service for instructions on how to do so. To add taints to an existing resource, it
is necessary to use ``kubectl``. Tolerations can be added to Pods by including the ``tolerations``
field in the Pod specification.

``kubectl`` Taints
------------------

To taint a node with kubectl, use ``kubectl taint nodes``.

.. code:: bash

   kubectl taint nodes ${NODE_NAME} ${TAINT_TYPE}=${TAINT_TAG}:${TAINT_EFFECT}

As an example, the following snippet taints nodes named ``node-1`` to not be schedulable if the
``accelerator`` taint type has the ``gpu`` taint value.

.. code:: bash

   kubectl taint nodes node-1 accelerator=gpu:NoSchedule

``kubectl`` Tolerations
-----------------------

To specify a toleration, use the ``toleration`` field in the PodSpec.

.. code:: yaml

   tolerations:
      - key: "${TAINT_TYPE}"
         operator: "Equal"
         value: "${TAINT_TAG}"
         effect: "${TAINT_EFFECT}"

The following example is a toleration for when a node has the ``accelerator`` taint type equal to
the ``gpu`` taint value.

.. code:: yaml

   tolerations:
      - key: "accelerator"
         operator: "Equal"
         value: "gpu"
         effect: "NoSchedule"

The next example is a toleration for when a node has the ``gpu`` taint type.

.. code:: yaml

   tolerations:
      - key: "gpu"
         operator: "Exists"
         effect: "NoSchedule"

.. _multi-rp-on-kubernetes:

Setting Up Multiple Resource Pools
==================================

To set up multiple resource pools for Determined on your Kubernetes cluster:

#. `Create a namespace
   <https://kubernetes.io/docs/tasks/administer-cluster/namespaces/#creating-a-new-namespace>`__ for
   each resource pool. The default namespace can also be mapped to a resource pool.

#. As Determined ensures that tasks in a given resource pool get launched in its linked namespace,
   the cluster admin needs to ensure that pods in a given namespace have the right nodeSelector or
   toleration automatically added to their pod spec so that they can be forced to be scheduled on
   the nodes that we want to be part of a given resource pool. This can be done using an admissions
   controller like a `PodNodeSelector
   <https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#podnodeselector>`__
   or `PodTolerationRestriction
   <https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#podtolerationrestriction>`__.
   Alternatively, the cluster admin can also add a resource pool (and hence namespace) specific pod
   spec to the ``task_container_defaults`` sub-section of the ``resourcePools`` section of the Helm
   ``values.yaml``:

   .. code:: yaml

      resourcePools:
        - pool_name: prod_pool
          kubernetes_namespace: default
          task_container_defaults:
            gpu_pod_spec:
              apiVersion: v1
              kind: Pod
              spec:
                tolerations:
                  - key: "pool_taint"
                    operator: "Equal"
                    value: "prod"
                    effect: "NoSchedule"

#. Label/taint the appropriate nodes you want to include as part of each resource pool. For instance
   you may add a taint like ``kubectl taint nodes prod_node_name pool_taint=prod:NoSchedule`` and
   the appropriate toleration to the PodTolerationRestriction admissions controller or in
   ``resourcePools.pool_name.task_container_defaults.gpu_pod_spec`` as above so it is automatically
   added to the pod spec based on which namespace (and hence resource pool) a task runs in.

#. Add the appropriate resource pool name to namespace mappings in the ``resourcePools`` section of
   the ``values.yaml`` file in the Helm chart.

********************
 Install Determined
********************

Once finished making configuration changes in ``values.yaml`` and ``Chart.yaml``, Determined is
ready to be installed. To install Determined, run:

.. code::

   helm install <name for your deployment> determined-helm-chart

``determined-helm-chart`` is a relative path to where the :download:`Determined Helm Chart
</helm/determined-latest.tgz>` is located. It may take a few minutes for all resources to come up.
If you encounter issues during installation, refer to the list of :ref:`useful kubectl commands
<useful-kubectl-commands>`. Helm will install Determined within the default namespace. If you wish
to install Determined into a non-default namespace, add ``-n <namespace name>`` to the command shown
above.

Once the installation has completed, instructions will be displayed for discovering the IP address
assigned to the Determined master. The IP address can also be discovered by running ``kubectl get
services``.

When installing Determined on Kubernetes, I get an ``ImagePullBackOff`` error
=============================================================================

You may be trying to install a non-released version of Determined or a version in a private registry
without the right secret. See the documentation on how to configure which :ref:`version of
Determined <configure-determined-kubernetes-version>` to install on Kubernetes.

********************
 Upgrade Determined
********************

To upgrade Determined or to change a configuration setting, first make the appropriate changes in
``values.yaml`` and ``Chart.yaml``, and then run:

.. code::

   helm upgrade <name for your deployment> --wait determined-helm-chart

Before upgrading Determined, consider pausing all active experiments. Any experiments that are
active when the Determined master restarts will resume training after the upgrade, but will be
rolled back to their most recent checkpoint.

**********************
 Uninstall Determined
**********************

To uninstall Determined run:

.. code::

   # Please note that if the Postgres Database was deployed by Determined, it will
   # be deleted by this command, permanently removing all records of your experiments.
   helm delete <name for your deployment>

   # If there were any active tasks when uninstalling, this command will
   # delete all of the leftover Kubernetes resources. It is recommended to
   # pause all experiments prior to upgrading or uninstalling Determined.
   kubectl get pods --no-headers=true -l=determined | awk '{print $1}' | xargs kubectl delete pod

************
 Next Steps
************

:doc:`custom-pod-specs` :doc:`k8s-dev-guide` :doc:`setup-aks-cluster` :doc:`setup-eks-cluster`
:doc:`setup-gke-cluster`
