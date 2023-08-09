:orphan:

.. _basic-setup:

############################
 Set Up Determined Overview
############################

.. meta::
   :description: Follow these steps to set up a brand new Determined training environment.

Using Determined requires a training environment. If you have not used Determined before, or you
want to set up a new training environment, you are at the right place!

This step-by-step checklist will help you get started by covering the basics of preparing for and
setting up a new training environment. After completing these steps, your users will be able to see
and access your Determined cluster.

.. note::

   Your training environment can be a local development machine, an on-premise GPU cluster, or cloud
   resources.

****************************
 Step 1 - Set Up PostgreSQL
****************************

Determined uses a PostgreSQL database to store experiment and trial metadata. Choose the
installation method that best fits your environment and requirements.

.. note::

   Kubernetes

   If you are using Kubernetes, you can skip this step. :ref:`Installing Determined on Kubernetes
   <determined-on-kubernetes>` uses the Determined Helm Chart which includes deployment of a
   PostgreSQL database.

   Cloud Services

   -  :ref:`AWS <install-aws>`. The Determined CLI manages the process of provisioning an Amazon RDS
      instance for PostgreSQL.
   -  :ref:`GCP <install-gcp>`. The Determined CLI manages the setup of Google Cloud SQL instances
      for PostgreSQL.


.. tabs::

   .. tab::

      Docker

      :ref:`Setting Up a Docker PostgreSQL Environment <install-postgres-docker>`.

   .. tab::

      Linux

      :ref:`Installing Determined using Linux Packages <install-using-linux-packages-preliminary>`
      pulls in the official Docker image for PostgreSQL.


****************************************
 Step 2 - Install the Determined Master
****************************************

The next step is to decide if you want to deploy the Determined Master on premises or on cloud.

.. tabs::

   .. tab::

      On Prem

      .. tabs::

         .. tab::

            Docker (Agent-Based)

            If the Determined Agent is your compute resource, you'll install the Determined Agent
            along with the Determined Master. The preferred method for installing the Agent is to
            use Linux packages. The recommended alternative to Linux packages is Docker.

            To install the Determined Master and Agent on premises, you'll first need to meet the
            installation requirements:

            -  :ref:`Installation Requirements <requirements>`

            Once you've met the installation requirements, select one of the following options:

            -  :ref:`Install Determined Using Docker <install-using-docker>`

         .. tab::

            Linux (Agent-Based)

            If the Determined Agent is your compute resource, you'll install the Determined Agent
            along with the Determined Master. The preferred method for installing the Agent is to
            use Linux packages. The recommended alternative to Linux packages is Docker.

            To install the Determined Master and Agent on premises, you'll first need to meet the
            installation requirements:

            -  :ref:`Installation Requirements <requirements>`

            Once you've met the installation requirements, select one of the following options:

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

It is recommended to use :ref:`Transport Layer Security (TLS) <tls>`. However, if you do not require
the secure version of HTTP, you can skip this section.

-  Master-Only TLS

Add instructions.

-  Mutual TLS

Agent-Based

In an agent-based installation, Determined is the resource manager.

To set up TLS for Agents, visit :ref:`Transport Security Layer--Agent Configuration
<tls-agent-config>`.

-  Kubernetes TLS

Add instructions.

*************************************
 Step 4 - Set Up Security (Optional)
*************************************

The next step is to configure your security features. Security is a shared responsibility between
you and Determined.

.. attention::

   Security features, with the exception of TLS, are only available on Determined Enterprise
   Edition.

.. tabs::

   .. tab::

      SSO

      .. tabs::

         .. tab::

            To Do Kubernetes

            To find out how to set up SSO with Kubernetes, visit :ref:`tls-agent-config`. .. _saml:

         .. tab::

            To Do Other

            To set up SSO in any environment other than Kubernetes, visit :ref:`tls-agent-config`.

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

RBAC
====

x

Workspaces
==========

x

Checkpoint Storage
==================

x

*********************
 Deploy Your Cluster
*********************

Once you have set up the necessary components for your chosen environment, you can configure the
environment. For detailed instructions by environment, visit the :ref:`Cluster Deployment Guide by
Environment <setup-checklists>`.
