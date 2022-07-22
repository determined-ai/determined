.. _custom-pod-specs:

#################
 Customize a Pod
#################

In a :ref:`Determined cluster running on Kubernetes <determined-on-kubernetes>`, tasks (e.g.,
experiments, notebooks) are executed by launching one or more Kubernetes pods. Users can customize
these pods by providing custom `pod specs
<https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#pod-v1-core>`__. Common use
cases include assigning pods to specific nodes, specifying additional volume mounts, and attaching
permissions. Configuring pod specs is not required to use Determined on Kubernetes.

In this topic guide, we will cover:

#. How Determined uses pod specs.
#. The different ways to configure custom pod specs.
#. Supported pod spec fields.
#. How to configuring default pod specs.
#. How to configuring per-task pods specs.

*******************************
 How Determined Uses Pod Specs
*******************************

All Determined tasks are launched as pods. Determined pods consists of an initContainer named
``determined-init-container`` and a container named ``determined-container`` which executes the
workload. When users provide a pod spec, Determined inserts the ``determined-init-container`` and
``determined-container`` into the provided pod spec. As described below, users may also configure
some of the fields for the ``determined-container``.

*****************************
 Ways to Configure Pod Specs
*****************************

Determined provides two ways to configure pod specs. When Determined is installed, the system
administrator can configure pod specs that are used by default for all GPU and CPU tasks. In
addition, users can specify a custom pod spec for individual tasks (e.g., for an experiment by
specifying ``environment.pod_spec`` in the :ref:`experiment configuration
<experiment-config-reference>`). If a custom pod spec is specified for a task, it overrides the
default pod spec (if any).

***************************
 Supported Pod Spec Fields
***************************

This section describes which fields users can and cannot configure when specifying custom `pod specs
<https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#pod-v1-core>`__.

Determined does not currently support configuring:

-  Pod Name - Determined automatically assigns a name for every pod that is created.
-  Pod Namespace - Determined launches all tasks in the Namespace in which the Determined master is
   running.
-  Host Networking - This must be configured via the :ref:`master-config-reference`.
-  Restart Policy - This is always set to ``Never``.

As part of their pod spec, users can specify ``initContainers`` and ``containers``. Additionally
users can configure the ``determined-container`` that executes the task (e.g., training), by setting
the container name in the pod spec to ``determined-container``. For the ``determined-container``,
Determined currently supports configuring:

-  Resource requests and limits (except GPU resources).
-  Volume mounts and volumes.

*******************
 Default Pod Specs
*******************

Default pod specs must be configured when :ref:`installing or upgrading <install-on-kubernetes>`
Determined. The default pod specs are configured in ``values.yaml`` of the
:doc:`/reference/reference-deploy/config/helm-config-reference` under
``taskContainerDefaults.cpuPodSpec`` and ``taskContainerDefaults.gpuPodSpec``. The ``gpuPodSpec`` is
applied to all tasks that use GPUs (e.g., experiments, notebooks). ``cpuPodSpec`` is applied to all
tasks that only use CPUs (e.g., TensorBoards, CPU-only notebooks). Fields that are not specified
will remain at their default Determined values.

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

.. _per-task-pod-specs:

********************
 Per-task Pod Specs
********************

In addition to default pod specs, it is also possible to configure custom pod specs for individual
tasks. When defining a custom pod spec for a task, it will override the default pod spec if one is
defined. Pod specs for individual tasks can be configured under the ``environment`` field in the
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
