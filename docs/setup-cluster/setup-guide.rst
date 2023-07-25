:orphan:

.. _basic-setup:

#############################################
 Checklist: Set Up Your Training Environment
#############################################

.. meta::
   :description: Follow these steps to set up a brand new Determined training environment.

Using Determined requires a training environment. If you have not used Determined before, or you
want to set up a new training environment, you are at the right place! This checklist will help you
get started by covering the basics of preparing for and setting up a new training environment.

.. note::

   Your training environment can be a local development machine, an on-premise GPU cluster, or cloud
   resources.

After completing the steps shown in this guide, your users will be able to see and access your
Determined cluster.

****************************
 Step 1 - Set Up PostgreSQL
****************************

The first step is to set up PostgreSQL. Determined uses a PostgreSQL database to store experiment
and trial metadata. Choose the installation method that best fits your environment and requirements.

.. note::

   If you are using Kubernetes, you can skip this step. :ref:`Installing Determined on Kubernetes
   <determined-on-kubernetes>` uses the Determined Helm Chart which includes deployment of a
   PostgreSQL database.

.. tabs::

   .. tab::

      Linux

      If you are installing Determined using Linux Packages, follow the :ref:`instructions
      <install-using-linux-packages-preliminary>`. Using this installation method pulls in the
      official Docker image for PostgreSQL.

   .. tab::

      Docker

      To set up PostgreSQL on Docker, follow the :ref:`instructions <install-postgres-docker>`.

   .. tab::

      Homebrew

      If you are installing Determined using Homebrew, follow the :ref:`instructions
      <install-using-homebrew-steps>`. Using this installation method pulls in postgreSQL as a
      dependency.

   .. tab::

      Cloud

      -  :ref:`AWS <install-aws>`. The Determined CLI manages the process of provisioning an Amazon
         RDS instance for PostgreSQL.
      -  :ref:`GCP <install-gcp>`. The Determined CLI manages the setup of Google Cloud SQL
         instances for PostgreSQL.

****************************************
 Step 2 - Install the Determined Master
****************************************

The next step is to decide if you want to deploy the Determined Master on premises or on cloud.

.. tabs::

   .. tab::

      On Prem

      .. tabs::

         .. tab::

            Agent-Based

            If the Determined Agent is your compute resource, you'll install the Determined Agent
            along with the Determined Master. The preferred method for installing the Agent is to
            use Linux packages. The recommended alternative to Linux packages is Docker.

            To install the Determined Master and Agent on premises, you'll first need to meet the
            installation requirements:

            -  :ref:`Installation Requirements <requirements>`

            Once you've met the installation requirements, select one of the following options:

            -  :ref:`Install Determined Using Docker <install-using-docker>`
            -  :ref:`Install Determined Using Linux Packages <install-using-linux-packages>`

         .. tab::

            Kubernetes

            To install the Determined Master on premises with Kubernetes, follow the steps below:

            -  :ref:`Deploy on Kubernetes <determined-on-kubernetes>`
            -  :ref:`Install Determined on Kubernetes <install-on-kubernetes>`

         .. tab::

            Slurm

            To install the Determined Master on premises with Slurm, follow the steps below:

            -  :ref:`sysadmin-deploy-on-hpc`

   .. tab::

      On Cloud

      .. tabs::

         .. tab::

            Agent-Based

            To install the Determined Master and Agent on cloud, select one of the following
            options:

            -  :ref:`AWS <install-aws>`
            -  :ref:`GCP <install-gcp>`

            .. note::

               When using AWS or GCP, ``det CLI`` manages the installation of the Determined Agent
               for you.

         .. tab::

            Kubernetes

            To install the Determined Master on cloud using Kubernetes, start here:

            -  :ref:`Install on Kubernetes <install-on-kubernetes>`

            After completing the step above, select one of the following options:

            -  :ref:`setup-eks-cluster`
            -  :ref:`setup-gke-cluster`
            -  :ref:`setup-aks-cluster`

********************************
 Step 3 - Set Up TLS (Optional)
********************************

It is recommended to use `Transport Layer Security (TLS) <_tls>`. However, if you do not require the
secure version of HTTP, you can skip this section.

.. tabs::

   .. tab::

      Agent-Based

      In an agent-based installation, Determined is the resource manager.

      To set up TLS for Agents, visit :ref:`Transport Security Layer--Agent Configuration
      <tls-agent-config>`.

   .. tab::

      Kubernetes

      To set up TLS on Kubernetes, choose one of the following methods:

      -  type here
      -  type here

   .. tab::

      Slurm

      To set up TLS on Slurm, (do something).

*************************************
 Step 4 - Set Up Security (Optional)
*************************************

To do: Add a sentence here describing why they would want to set up security.

.. attention::

   SSO is only supported on the Determined Enterprise Edition.

.. tabs::

   .. tab::

      Kubernetes

      To set up SSO, follow these instructions:

      -  x
      -  x
      -  x

   .. tab::

      Other

      To set up security in any environment other than Kubernetes, (do something).

To validate Step 4, ensure the users can access the Determined cluster.

***********************************
 Step 5 - Set Up Compute Resources
***********************************

.. tabs::

   .. tab::

      Linux Packages

      Description and link to instructions.

   .. tab::

      Docker

      Description and link to instructions.

   .. tab::

      Kubernetes

      Description and link to instructions.

   .. tab::

      Slurm

      Description and link to instructions.

*********************************************
 Step 6 - Set Up Monitoring Tools (Optional)
*********************************************

The following monitoring tools are officially supported: Prometheus/Grafana

.. tabs::

   .. tab::

      Prometheus

      Description and link to instructions.

   .. tab::

      Grafana

      Description and link to instructions.

************
 Next Steps
************

To find out how to manage your training environment, visit the :ref:`Cluster Deployment Guide by
Environment <setup-checklists>` and follow the steps shown for your environment.

RBAC
====

x

Workspaces
==========

x

Checkpoint Storage
==================

x

Deploy Your Cluster
===================

Once you have set up the necessary components for your chosen environment, you can configure the
environment. For detailed instructions by environment, visit the :ref:`Cluster Deployment Guide by
Environment <setup-checklists>`.
