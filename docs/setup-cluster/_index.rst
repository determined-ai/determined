.. _installation-guide:

###############################
 Install and Set Up Determined
###############################

.. meta::
   :description: Discover how to install and set up your Determined cluster locally, on AWS, on GCP, on Kubernetes, on Slurm or on premises.

Install and set up Determined using the cluster deployment guide for your environment.

.. include:: ../_shared/tip-keep-install-instructions.txt

.. note::

   To configure your cluster, visit :ref:`cluster-configuration`.

*******
 Local
*******

Install Determined on a single machine, for your own use. Compatible with Windows, Mac, and Linux.
Ideal for getting started with Determined.

-  :ref:`basic`
-  :ref:`install-using-deploy`
-  :ref:`install-using-homebrew`
-  :ref:`install-using-wsl`

******************
 Determined Agent
******************

Use Determined’s built-in resource management. This is an easier alternative to installing and
administering via Kubernetes or Slurm. Ideal for teams of any size to share dedicated compute
resources. Compatible with on-prem clusters and cloud auto-scaling (AWS and GCP).

-  :ref:`deploy-on-prem-overview`

   -  :ref:`install-using-linux-packages`
   -  :ref:`install-using-docker`

-  :ref:`topic_guide_aws`

-  :ref:`topic_guide_gcp`

************
 Kubernetes
************

Allow Determined to submit jobs to a Kubernetes cluster. Compatible with on-prem, GKE, and EKS
clusters.

-  :ref:`determined-on-kubernetes`

   -  :ref:`install-on-kubernetes`
   -  :ref:`setup-aks-cluster`
   -  :ref:`setup-eks-cluster`
   -  :ref:`setup-gke-cluster`

*******
 Slurm
*******

Enable Determined to submit jobs to a Slurm cluster.

.. attention::

   This method is only available on Determined Enterprise Edition.

-  :ref:`sysadmin-deploy-on-hpc`

.. toctree::
   :hidden:

   Quick Installation <basic>
   Deploy on Prem <on-prem/_index>
   Deploy on AWS <aws/_index>
   Deploy on GCP <gcp/_index>
   Deploy on Kubernetes <k8s/_index>
   Deploy on Slurm/PBS <slurm/_index>
   Cluster Configuration <cluster-configuration>
