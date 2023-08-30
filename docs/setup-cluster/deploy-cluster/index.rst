.. _installation-guide:

###############################
 Install and Set Up Determined
###############################

.. meta::
   :description: Discover how to install and set up your Determined cluster locally, on AWS, on GCP, on Kubernetes, on Slurm or on premises.

To install and set up Determined, follow the cluster deployment guide for your environment. For
additional installation options, visit :ref:`installation-options`.

+--------------------------------------------------------+
| Local                                                  |
+========================================================+
| -  :doc:`basic`                                        |
| -  :doc:`on-prem/deploy`                               |
| -  :doc:`on-prem/homebrew`                             |
| -  :doc:`on-prem/wsl`                                  |
+--------------------------------------------------------+

+--------------------------------------------------------+
| Determined Agent                                       |
+========================================================+
| :doc:`on-prem/overview`                                |
|                                                        |
| -  :doc:`on-prem/linux-packages`                       |
| -  :doc:`on-prem/docker`                               |
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

.. note::

   For more configuration options, visit :ref:`cluster-configuration`.

.. toctree::
   :hidden:

   Basic Installation <basic>
   Deploy on Prem <on-prem/overview>
   Deploy on AWS <aws/overview>
   Deploy on GCP <gcp/overview>
   Deploy on Kubernetes <k8s/overview>
   Deploy on Slurm/PBS <slurm/overview>
   Installation Options <installation-options>
