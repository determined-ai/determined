.. _k8s-helm-reference:

###################################
 Helm and Kubectl Command Examples
###################################

+-----------------------------------------------------------------+
| Configuration Reference                                         |
+=================================================================+
| :ref:`helm-config-reference`                                    |
+-----------------------------------------------------------------+

`kubectl <https://kubernetes.io/docs/tasks/tools/>`_ is a command-line tool for interacting with a
Kubernetes cluster. `Helm <https://helm.sh/docs/helm/helm_install/>`_ is used to install and upgrade
Determined on Kubernetes. This section covers some of the useful kubectl and helm commands when
:ref:`running Determined on Kubernetes <install-on-kubernetes>`.

For all the commands listed below, include ``-n <kubernetes namespace name>`` if running Determined
in a non-default `namespace
<https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/>`_.

*******************************
 List Determined Installations
*******************************

To list the current installation of Determined on the Kubernetes cluster:

.. code:: bash

   # To list in the current namespace.
   helm list

   # To list in all namespaces.
   helm list -A

It is recommended to have just one instance of Determined per Kubernetes cluster.

****************************
 AMD ROCm GPU Configuration
****************************

For specific configuration details related to AMD ROCm GPUs, including how to set up resource pools
and configure experiments, see our :ref:`guide on Configuring Kubernetes for ROCm GPUs
<rocm-config-k8s>`.

**************************************
 Get the Determined Master IP Address
**************************************

To get the IP and port address of the Determined master:

.. code:: bash

   # Get all services.
   kubectl get services

   # Get the master service. The exact name of the master service depends on
   # the name given to your helm deployment, which can be looked up by running
   # ``helm list``.
   kubectl get service determined-master-service-<helm deployment name>

************************************
 Check the Determined Master Status
************************************

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

*********************************************
 Get All Determined-launched Kubernetes Jobs
*********************************************

On Determined with Kubernetes, tasks start their own jobs, which have associated pods. These
``kubectl`` commands list and delete pods which are running Determined tasks:

.. code:: bash

   # Get all jobs that are running Determined tasks.
   kubectl get jobs -l=determined

   # Get all pods associated with a given job.
   kubectl get pods -l="batch.kubernetes.io/job-name=<determined-job-name>"

   # Delete all Determined jobs for all tasks for ALL clusters. Users should never have to run this,
   # unless they are removing a deployment of Determined.
   kubectl get jobs --no-headers=true -l=determined | awk '{print $1}' | xargs kubectl delete jobs

   # Get logs for a Determined task that make it to STDOUT or STDERR. Most logs are shipped to the
   # Determined API server but logs that can't be shipped still go here. This is useful for debugging
   # log shipping failures.
   # For Determined tasks that require multiple pods, this will return logs for only one pod. It is
   # recommended that you search the logs for each pod individually.
   kubectl logs jobs/<determined-job-name>

***************************
 Useful Debugging Commands
***************************

.. code:: bash

   # Get the name of the Helm deployment.
   helm list

   # Double check the IP address and port assigned to the Determined master by looking up the master service.
   kubectl get service determined-master-service-development-<helm deployment name>

   # Check the status of master deployment.
   kubectl describe deployment determined-master-deployment-<helm deployment name>

   # Check the logs of master pod.
   kubectl logs <determined-master-pod-name>
