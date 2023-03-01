###################
 Set Up Determined
###################

To set up Determined, start by following the cluster deployment guide for your environment.

+------------------------------------------+-----------------------------------------------------+
| Environment                              | Description                                         |
+==========================================+=====================================================+
| :doc:`sysadmin-deploy-on-prem/overview`  | How to deploy Determined on-premises.               |
+------------------------------------------+-----------------------------------------------------+
| :doc:`sysadmin-deploy-on-aws/overview`   | How to deploy Determined on Amazon Web Services.    |
+------------------------------------------+-----------------------------------------------------+
| :doc:`sysadmin-deploy-on-gcp/overview`   | How to deploy Determined on Google Cloud Platform.  |
+------------------------------------------+-----------------------------------------------------+
| :doc:`sysadmin-deploy-on-k8s/overview`   | How to run Determined on Kubernetes.                |
+------------------------------------------+-----------------------------------------------------+
| :doc:`sysadmin-deploy-on-slurm/overview` | How to run Determined on an HPC cluster             |
|                                          | (Slurm/PBS).                                        |
+------------------------------------------+-----------------------------------------------------+

*************
 Basic Setup
*************

Your training environment can be a local development machine, an on-premise GPU cluster, or cloud
resources. To set up your training environment, follow the :doc:`Basic Setup
</cluster-setup-guide/basic>` guide.

*******************************
 Installing the Determined CLI
*******************************

The Determined CLI is a command line tool that lets you launch new experiments and interact with a
Determined cluster. The CLI can be installed on any machine you want to use to access Determined. To
install the CLI, follow the :ref:`install-cli` instructions.

************************************
 Configuring the Determined Cluster
************************************

-  :doc:`Common configuration options </reference/reference-deploy/config/common-config-options>`
-  :doc:`Master configuration reference
   </reference/reference-deploy/config/master-config-reference>`
-  :doc:`Agent configuration reference </reference/reference-deploy/config/agent-config-reference>`

.. toctree::
   :hidden:

   Deploy on Prem <sysadmin-deploy-on-prem/overview>
   Deploy on AWS <sysadmin-deploy-on-aws/overview>
   Deploy on GCP <sysadmin-deploy-on-gcp/overview>
   Deploy on Kubernetes <sysadmin-deploy-on-k8s/overview>
   Deploy on Slurm/PBS <sysadmin-deploy-on-slurm/overview>
