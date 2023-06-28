.. _setup-checklists:

#########################################
 Cluster Deployment Guide by Environment
#########################################

.. meta::
   :description: Discover how to set up your Determined cluster on AWS, GCP, Kubernetes, Slurm or On-Prem with an easy checklist.

To set up Determined, start by following the :ref:`basic setup guide <basic-setup>` then consult one
of the checklists below.

+--------------------------------------------------------+
| Environment                                            |
+========================================================+
| :doc:`on-prem/overview`                                |
|                                                        |
| -  :doc:`on-prem/linux-packages`                       |
| -  :doc:`on-prem/deploy`                               |
| -  :doc:`on-prem/docker`                               |
| -  :doc:`on-prem/homebrew`                             |
| -  :doc:`on-prem/wsl`                                  |
+--------------------------------------------------------+
| :doc:`aws/overview`                                    |
+--------------------------------------------------------+
| :doc:`gcp/overview`                                    |
+--------------------------------------------------------+
| :doc:`k8s/overview`                                    |
|                                                        |
| -  :doc:`k8s/install-on-kubernetes`                    |
| -  :doc:`k8s/setup-aks-cluster`                        |
| -  :doc:`k8s/setup-eks-cluster`                        |
| -  :doc:`k8s/setup-gke-cluster`                        |
+--------------------------------------------------------+
| :doc:`slurm/overview`                                  |
+--------------------------------------------------------+

************************************
 Configuring the Determined Cluster
************************************

-  :doc:`Common configuration options </reference/deploy/config/common-config-options>`
-  :doc:`Master configuration reference </reference/deploy/config/master-config-reference>`
-  :doc:`Agent configuration reference </reference/deploy/config/agent-config-reference>`

.. toctree::
   :hidden:

   Basic Setup Guide <../basic-setup>
   Deploy on Prem <on-prem/overview>
   Deploy on AWS <aws/overview>
   Deploy on GCP <gcp/overview>
   Deploy on Kubernetes <k8s/overview>
   Deploy on Slurm/PBS <slurm/overview>
