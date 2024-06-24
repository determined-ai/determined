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

.. note::

   Administrators should first consult :ref:`advanced-setup-checklist` when setting up Determined
   for an organization.

*******
 Local
*******

Install Determined on a single machine, for your own use. Compatible with Windows, Mac, and Linux.
Ideal for getting started with Determined.

-  :ref:`basic`
-  :ref:`install-using-deploy`
-  :ref:`install-using-homebrew`
-  :ref:`install-using-wsl`

.. note::

   **Initial Password**: When you deploy locally using ``det deploy local`` with ``master-up`` or ``cluster-up`` commands
   and no user accounts have been created yet, an initial password will be automatically generated
   and shown to the user (with the option to change it) if neither
   ``security.initial_user_password`` in ``master.yaml`` nor the ``--initial-user-password`` CLI
   flag is present.

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

-  :ref:`topic_guide_mldes`

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

   Quick Installation <../get-started/basic>
   Advanced Installation <checklists/_index>
   Deploy via MLDES <mldes/_index>
   Deploy on Prem <on-prem/_index>
   Deploy on AWS <aws/_index>
   Deploy on GCP <gcp/_index>
   Deploy on Kubernetes <k8s/_index>
   Deploy on Slurm/PBS <slurm/_index>
   Cluster Configuration <cluster-configuration>
