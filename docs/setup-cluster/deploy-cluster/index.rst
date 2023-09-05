.. _installation-guide:

###############################
 Install and Set Up Determined
###############################

.. meta::
   :description: Discover how to install and set up your Determined cluster locally, on AWS, on GCP, on Kubernetes, on Slurm or on premises.

To install and set up Determined, follow the cluster deployment guide for your environment.

.. note::

   To configure your cluster, visit :ref:`cluster-configuration`.

*******
 Local
*******

Run Determined on your own machine for development or testing purposes.

-  :ref:`basic`
-  :ref:`install-using-deploy`
-  :ref:`install-using-homebrew`
-  :ref:`install-using-wsl`

******************
 Determined Agent
******************

Manage compute resources and oversee training tasks through Determined agents for scalability in
diverse environments.

-  :ref:`deploy-on-prem-overview`

   -  :ref:`install-using-linux-packages`
   -  :ref:`install-using-docker`

-  :ref:`topic_guide_aws`

-  :ref:`topic_guide_gcp`

************
 Kubernetes
************

-  :ref:`determined-on-kubernetes`

   -  :ref:`install-on-kubernetes`
   -  :ref:`setup-aks-cluster`
   -  :ref:`setup-eks-cluster`
   -  :ref:`setup-gke-cluster`

*******
 Slurm
*******

-  :ref:`sysadmin-deploy-on-hpc`

.. toctree::
   :hidden:

   Quick Installation <basic>
   Deploy on Prem <on-prem/overview>
   Deploy on AWS <aws/overview>
   Deploy on GCP <gcp/overview>
   Deploy on Kubernetes <k8s/overview>
   Deploy on Slurm/PBS <slurm/overview>
   Cluster Configuration <cluster-configuration>
