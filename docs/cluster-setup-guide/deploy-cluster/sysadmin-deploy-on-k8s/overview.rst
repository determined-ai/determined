.. _determined-on-kubernetes:

######################
 Deploy on Kubernetes
######################

This document describes how the Determined runs on `Kubernetes <https://kubernetes.io/>`__. For
instructions on installing Determined on Kubernetes, see the :ref:`installation guide
<install-on-kubernetes>`.

In this topic guide, we will cover:

#. How Determined works on Kubernetes.
#. Limitations of Determined on Kubernetes.
#. Useful Helm and Kubectl commands.

************************************
 How Determined Works on Kubernetes
************************************

:ref:`Installing Determined on Kubernetes <install-on-kubernetes>` deploys an instance of the
Determined master and a Postgres database in the Kubernetes cluster. Once the master is up and
running, you can launch :ref:`experiments <experiments>`, :ref:`notebooks <notebooks>`,
:ref:`TensorBoards <tensorboards>`, :ref:`commands <commands-and-shells>`, and :ref:`shells
<commands-and-shells>`. When new workloads are submitted to the Determined master, the master
launches pods and configMaps on the Kubernetes cluster to execute those workloads. Users of
Determined shouldn't need to interact with Kubernetes directly after installation, as Determined
handles all the necessary interaction with the Kubernetes cluster.

It is also important to note that when running Determined on Kubernetes, a higher priority value
means a higher priority (e.g. a priority 50 task will run before a priority 40 task). This is
different from priority scheduling in non-Kubernetes deployments, where lower priority values mean a
higher priority (e.g. a priority 40 task will run before a priority 50 task).

.. _limitations-on-kubernetes:

***************************
 Limitations on Kubernetes
***************************

This section outlines the current limitations of Determined on Kubernetes.

Scheduling
==========

By default, the Kubernetes scheduler does not support gang scheduling or preemption. This can be
problematic for distributed deep learning workloads that require multiple pods to be scheduled
before execution starts. Determined includes built-in support for the `lightweight coscheduling
plugin <https://github.com/kubernetes-sigs/scheduler-plugins/tree/release-1.18/pkg/coscheduling>`__,
which extends the default Kubernetes scheduler to support gang scheduling. Determined also includes
support for priority-based preemption scheduling. Neither are enabled by default. For more details
and instructions on how to enable the coscheduling plugin, refer to
:ref:`gang-scheduling-on-kubernetes` and :ref:`priority-scheduling-on-kubernetes`.

Dynamic Agents
==============

Determined is not able to autoscale your cluster, but equivalent functionality is available by using
the `Kubernetes Cluster Autoscaler
<https://github.com/kubernetes/autoscaler/tree/master/cluster-autoscaler>`_, which is supported on
`GKE <https://cloud.google.com/kubernetes-engine/docs/concepts/cluster-autoscaler>`_ and `EKS
<https://docs.aws.amazon.com/eks/latest/userguide/cluster-autoscaler.html>`_.

Pod Security
============

By default, Determined runs task containers as root. However, it is possible to associate a
Determined user with a Unix user and group, provided that the Unix user and group already exist.
Tasks initiated by the associated Determined user will run under the linked Unix user rather than
root. For more information, see: :ref:`run-as-user`.

.. _useful-kubectl-commands:

**********************************
 Useful Helm and Kubectl Commands
**********************************

`kubectl <https://kubernetes.io/docs/tasks/tools/install-kubectl/>`_ is a command-line tool for
interacting with a Kubernetes cluster. `Helm <https://helm.sh/docs/helm/helm_install/>`_ is used to
install and upgrade Determined on Kubernetes. This section covers some of the useful kubectl and
helm commands when :ref:`running Determined on Kubernetes <install-on-kubernetes>`.

For all the commands listed below, include ``-n <kubernetes namespace name>`` if running Determined
in a non-default `namespace
<https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/>`_.

List Installations of Determined
================================

To list the current installation of Determined on the Kubernetes cluster:

.. code:: bash

   # To list in the current namespace.
   helm list

   # To list in all namespaces.
   helm list -A

It is recommended to have just one instance of Determined per Kubernetes cluster.

Get the IP Address of the Determined Master
===========================================

To get the IP and port address of the Determined master:

.. code:: bash

   # Get all services.
   kubectl get services

   # Get the master service. The exact name of the master service depends on
   # the name given to your helm deployment, which can be looked up by running
   # ``helm list``.
   kubectl get service determined-master-service-<helm deployment name>

Check the Status of the Determined Master
=========================================

Logs for the Determined master are available via the CLI and WebUI. ``Kubectl`` commands are useful
for diagnosing any issues that arise during installation.

.. code:: bash

   # Get all deployments.
   kubectl get deployments

   # Describe the current state of Determined master deployment. The exact name
   # of the master deployment depends on the name given to your helm deploy
   # which can be looked up by running `helm list`.
   kubectl describe deployment determined-master-deployment-<helm deployment name>

   # Get all pods associated with the Determined master deployment. Note this
   # will only include pods that are running the Determined master, not pods
   # that are running tasks associated with Determined workloads.
   kubectl get pods -l=app=determined-master-<helm deployment name>

   # Get logs for the pod running the Determined master.
   kubectl logs <determined-master-pod-name>

Get All Running Task Pods
=========================

These ``kubectl`` commands list and delete pods which are running Determined tasks:

.. code:: bash

   # Get all pods that are running Determined tasks.
   kubectl get pods -l=determined

   # Delete all Determined task pods. Users should never have to run this,
   # unless they are removing a deployment of Determined.
   kubectl get pods --no-headers=true -l=determined | awk '{print $1}' | xargs kubectl delete pod

.. toctree::
   :maxdepth: 1
   :hidden:

   install-on-kubernetes
   setup-aks-cluster
   setup-eks-cluster
   setup-gke-cluster
   k8s-dev-guide
   custom-pod-specs
   helm-commands
   troubleshooting
