.. _custom-pod-specs:

#################
 Customize a Pod
#################

In a :ref:`Determined cluster running on Kubernetes <determined-on-kubernetes>`, tasks (e.g.,
experiments, notebooks) are executed by launching one or more Kubernetes pods. You can customize
these pods by providing custom `pod specs
<https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#pod-v1-core>`__. Common use
cases include assigning pods to specific nodes, specifying additional volume mounts, and attaching
permissions. Configuring pod specs is not required to use Determined on Kubernetes.

In this topic guide, we will cover:

#. How Determined uses pod specs.
#. The different ways to configure custom pod specs.
#. Supported pod spec fields.
#. How to configure default pod specs.
#. How to configure per-task pods specs.

*******************************
 How Determined Uses Pod Specs
*******************************

All Determined tasks are launched as pods. Determined pods consists of an initContainer named
``determined-init-container`` and a container named ``determined-container`` which executes the
workload. When you provide a pod spec, Determined inserts the ``determined-init-container`` and
``determined-container`` into the provided pod spec. You may also configure some of the fields for
the ``determined-container``, as described below.

*****************************
 Ways to Configure Pod Specs
*****************************

Determined provides two ways to configure pod specs. When Determined is installed, the system
administrator can configure pod specs that are used by default for all GPU and CPU tasks. In
addition, you can specify a custom pod spec for individual tasks (e.g., for an experiment by
specifying ``environment.pod_spec`` in the :ref:`experiment configuration
<experiment-config-reference>`). If a custom pod spec is specified for a task, it overrides the
default pod spec (if any).

***************************
 Supported Pod Spec Fields
***************************

This section describes which fields can and cannot be configured when specifying custom `pod specs
<https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#pod-v1-core>`__.

Not Supported
=============

Determined does not support configuring the following fields:

-  Pod Name - Determined automatically assigns a name for every pod that is created.

-  Pod Namespace - Determined automatically sets the pod namespace based on the resource pool the
   task belongs to. The mapping between resource pools and namespaces can be configured in the
   ``resourcePools`` section of the Helm ``values.yaml``.

-  Host Networking - This must be configured via the :ref:`master configuration
   <master-config-reference>`.

-  Restart Policy - This is always set to ``Never``.

Supported
=========

As part of your pod spec, you can specify ``initContainers`` and ``containers``. Additionally you
can configure the ``determined-container`` that executes the task (e.g., training), by setting the
container name in the pod spec to ``determined-container``. For the ``determined-container``,

Determined supports configuring the following fields:

-  Resource requests and limits (except GPU resources).

-  Volume mounts and volumes.

-  All ``securityContext`` fields within the pod spec of the ``determined-container`` container
   except for ``RunAsUser`` and ``RunAsGroup``.

   For those fields, use ``det user link-with-agent-user`` instead.

   Example of configuring a pachyderm notebook plugin to run in ``det notebook``:

   .. code:: yaml

      environment:
        pod_spec:
          apiVersion: v1
          kind: Pod
          spec:
            containers:
              - name: determined-container
                  securityContext:
                    privileged: true

*******************
 Default Pod Specs
*******************

Default pod specs must be configured when :ref:`installing or upgrading <install-on-kubernetes>`
Determined. The default pod specs are configured in ``values.yaml`` of the
:doc:`/reference/deploy/helm-config-reference` under ``taskContainerDefaults.cpuPodSpec`` and
``taskContainerDefaults.gpuPodSpec``. The ``gpuPodSpec`` is applied to all tasks that use GPUs
(e.g., experiments, notebooks). ``cpuPodSpec`` is applied to all tasks that only use CPUs (e.g.,
TensorBoards, CPU-only notebooks). Fields that are not specified will remain at their default
Determined values.

Example of configuring default pod specs in ``values.yaml``:

.. code:: yaml

   taskContainerDefaults:
     cpuPodSpec:
       apiVersion: v1
       kind: Pod
       metadata:
         labels:
           customLabel: cpu-label
       spec:
         containers:
           # Will be applied to the container executing the task.
           - name: determined-container
             volumeMounts:
               - name: example-volume
                 mountPath: /example-data
           # Custom sidecar container.
           - name: sidecar-container
             image: alpine:latest
         volumes:
           - name: example-volume
             hostPath:
               path: /data
     gpuPodSpec:
       apiVersion: v1
       kind: Pod
       metadata:
         labels:
           customLabel: gpu-label
       spec:
         containers:
           - name: determined-container
             volumeMounts:
               - name: example-volume
                 mountPath: /example-data
         volumes:
           - name: example-volume
             hostPath:
               path: /data

The default pod specs can also be configured on a resource pool level. GPU jobs submitted in the
resource pool will have the task spec applied. If a job is submitted in a resource pool with a
matching CPU / GPU pod spec then the top level ``taskContainerDefaults.gpuPodSpec`` or
``taskContainerDefaults.cpuPodSpec`` will not be applied.

Example of configuring resource pool default pod spec in ``values.yaml``.

.. code:: yaml

   resourcePools:
     - pool_name: prod_pool
       kubernetes_namespace: default
       task_container_defaults:
         gpu_pod_spec:
           apiVersion: v1
           kind: Pod
           spec:
             affinity:
               nodeAffinity:
                 requiredDuringSchedulingIgnoredDuringExecution:
                   nodeSelectorTerms:
                     - matchExpressions:
                         - key: topology.kubernetes.io/zone
                           operator: In
                           values:
                             - antarctica-west1

.. _per-task-pod-specs:

********************
 Per-task Pod Specs
********************

In addition to default pod specs, it is also possible to configure custom pod specs for individual
tasks. Pod specs for individual tasks can be configured under the ``environment`` field in the
:ref:`experiment config <exp-environment>` (for experiments) or the :ref:`task configuration
<command-notebook-configuration>` (for other tasks).

Example of configuring a pod spec for an individual task:

.. code:: yaml

   environment:
     pod_spec:
       apiVersion: v1
       kind: Pod
       metadata:
         labels:
           customLabel: task-specific-label
       spec:
         # Specify a pull secret for task container image.
         imagePullSecrets:
           - name: regcred
         # Specify a service account that allows writing checkpoints to S3 (for EKS).
         serviceAccountName: <checkpoint-storage-s3-bucket>
         # Specify tolerations for scheduling on tainted nodes.
         tolerations:
           - key: "tained-nodegroup-name"
             operator: "Equal"
             value: "true"
             effect: "NoSchedule"

When a custom pod spec is provided for a task, it will merge with the default pod spec (either
``resourcePools.task_container_defaults`` or top level ``task_container_defaults`` if
``resourcePools.task_container_defaults`` is not specified) according to Kubernetes `strategic merge
patch
<https://kubernetes.io/docs/tasks/manage-kubernetes-objects/update-api-object-kubectl-patch/#use-a-strategic-merge-patch-to-update-a-deployment>`__.
Determined does not support setting the strategic merge patch strategy, so the section titled "Use
strategic merge patch to update a Deployment using the retainKeys strategy" in the linked Kubernetes
docs will not work.

Some fields in pod specs are merged by values of items in lists. Volumes for example are merged by
volume name. If for some reason you would want to remove a volume mount specific in the default task
container you would need to override it with an empty volume of the same path.

Example ``values.yaml``

.. code:: yaml

   resourcePools:
     - pool_name: prod_pool
       kubernetes_namespace: default
       task_container_defaults:
         gpu_pod_spec:
           apiVersion: v1
           kind: Pod
           spec:
             volumes:
               - name: secret-volume
                 secret:
                   secretName: prod-test-secret
             containers:
               - name: determined-container
                 volumeMounts:
                   - name: secret-volume
                     mountPath: /etc/secret-volume

Example ``expconf.yaml``

.. code:: yaml

   environment:
     pod_spec:
       apiVersion: v1
       kind: Pod
       spec:
         volumes:
           - name: empty-dir-override
             emptyDir:
               sizeLimit: 100Mi
         containers:
           - name: determined-container
             volumeMounts:
               - name: empty-dir-override
                 mountPath: /etc/secret-volume
   resources:
     resource_pool: prod_pool
