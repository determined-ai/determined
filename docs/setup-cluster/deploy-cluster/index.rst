.. _setup-checklists:

##########################
 Cluster Deployment Guide
##########################

.. meta::
   :description: Discover how to set up your Determined cluster on AWS, GCP, Kubernetes, Slurm or On-Prem with an easy checklist.

If you have not yet set up your training environment, consult the :ref:`setup checklist
<basic-setup>`.

+--------------------------------------------------------+
| Determined Agent                                       |
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

+--------------------------------------------------------+
| Kubernetes                                             |
+========================================================+
| :doc:`k8s/overview`                                    |
|                                                        |
| -  :doc:`k8s/install-on-kubernetes`                    |
| -  :doc:`k8s/setup-aks-cluster`                        |
| -  :doc:`k8s/setup-eks-cluster`                        |
| -  :doc:`k8s/setup-gke-cluster`                        |
+--------------------------------------------------------+

+--------------------------------------------------------+
| Slurm                                                  |
+========================================================+
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

   Basic Setup <../basic>
   Advanced Setup <setup-guide/overview>
   By Environment <self>
   Deploy on Prem <on-prem/overview>
   Deploy on AWS <aws/overview>
   Deploy on GCP <gcp/overview>
   Deploy on Kubernetes <k8s/overview>
   Deploy on Slurm/PBS <slurm/overview>
