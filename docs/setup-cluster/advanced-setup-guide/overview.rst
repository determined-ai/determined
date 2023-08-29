.. _advanced-setup:

################
 Advanced Setup
################

.. meta::
   :description: Follow these steps to set up a Determined training environment on-prem or on cloud.

Using Determined requires a training environment. Your training environment can be a local
development machine, an on-premise GPU cluster, or cloud resources.

This step-by-step checklist will help you get started by covering the basics of preparing for and
setting up a new training environment. After completing these steps, your users will be able to see
and access your Determined cluster.

.. note::

   To find out how to quickly set up a local training environment, visit :ref:`Basic Setup
   <basic-setup>`.

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

The next step is to decide if you want to deploy the Determined master on premises or on cloud.

.. tabs::

   .. tab::

      On Prem

      .. tabs::

         .. tab::

            Docker (Agent-Based)

            If the Determined agent is your compute resource, you'll install the Determined agent
            along with the Determined master. The preferred method for installing the agent is to
            use Linux packages. The recommended alternative to Linux packages is Docker.

            To install the Determined master and agent on premises, you'll first need to meet the
            installation requirements:

            -  :ref:`Installation Requirements <requirements>`

            Once you've met the installation requirements, select one of the following options:

            -  :ref:`Install Determined Using Docker <install-using-docker>`

         .. tab::

            Linux (Agent-Based)

            If the Determined agent is your compute resource, you'll install the Determined agent
            along with the Determined master. The preferred method for installing the Agent is to
            use Linux packages. The recommended alternative to Linux packages is Docker.

            To install the Determined master and agent on premises, you'll first need to meet the
            installation requirements:

            -  :ref:`Installation Requirements <requirements>`

            Once you've met the installation requirements, select one of the following options:

            -  :ref:`Install Determined Using Linux Packages <install-using-linux-packages>`

         .. tab::

            Kubernetes

            To install the Determined master on premises with Kubernetes, follow the steps below:

            -  :ref:`Deploy on Kubernetes <determined-on-kubernetes>`
            -  :ref:`Install Determined on Kubernetes <install-on-kubernetes>`

         .. tab::

            Slurm

            To install the Determined master on premises with Slurm, follow the steps below:

            -  :ref:`sysadmin-deploy-on-hpc`

   .. tab::

      On Cloud

      .. tabs::

         .. tab::

            Agent-Based

            To install the Determined master and agent on cloud, select one of the following
            options:

            -  :ref:`AWS <install-aws>`
            -  :ref:`GCP <install-gcp>`

            .. note::

               When using AWS or GCP, ``det CLI`` manages the installation of the Determined agent
               for you.

         .. tab::

            Kubernetes

            To install the Determined master on cloud using Kubernetes, start here:

            -  :ref:`Install on Kubernetes <install-on-kubernetes>`

            After completing the step above, select one of the following options:

            -  :ref:`setup-eks-cluster`
            -  :ref:`setup-gke-cluster`
            -  :ref:`setup-aks-cluster`

**Firewall Rules**

The :ref:`firewall rules <firewall-rules>` must satisfy network access requirements.

Configure Your Cluster
======================

Once you have set up the necessary components for your environment, configure the cluster.

Visit the cluster configuration resources below or visit :ref:`cluster-configuration`.

-  Common configuration reference: :doc:`/reference/deploy/config/common-config-options`
-  Master configuration reference: :doc:`/reference/deploy/config/master-config-reference`
-  Agent configuration reference: :doc:`/reference/deploy/config/agent-config-reference`

********************************
 Step 3 - Set Up TLS (Optional)
********************************

It is recommended to use :ref:`Transport Layer Security (TLS) <tls>`. However, if you do not require
the secure version of HTTP, you can skip this section.

Master-Only TLS
===============

:ref:`Transport Layer Security (TLS) <tls>`

Mutual TLS
==========

:ref:`Transport Layer Security (TLS) <tls>`

Agent-Based
===========

In an agent-based installation, Determined is the resource manager. To set up TLS for Agents, visit
:ref:`Transport Security Layer--Agent Configuration <tls-agent-config>`.

Kubernetes TLS
==============

:ref:`tls-on-kubernetes`

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

            Kubernetes

            To find out how to set up SSO with Kubernetes, visit :ref:`tls-agent-config`. .. _saml:

         .. tab::

            Other

            To set up SSO in any environment other than Kubernetes, visit :ref:`tls-agent-config`.

To validate Step 4, ensure the users can access the Determined cluster.

***********************************
 Step 5 - Set Up Compute Resources
***********************************

Set up your compute resources (such as Determined agents) to communicate with the Determined master.

-  :ref:`Firewall rules <firewall-rules>` must satisfy network access requirements for the master
      and agents.
-  :ref:`internet-access`
-  :ref:`setup-clients`

*********************************************
 Step 6 - Set Up Monitoring Tools (Optional)
*********************************************

To set up your monitoring tools, visit :ref:`Configure Determined with Prometheus and Grafana
<prometheus-grafana>`.

************
 Next Steps
************

Once you have completed the steps in this checklist, your users should be able to see and connect to
the Determined master.

Here are some additional steps to consider:

Configure RBAC
==============

You should configure role-based access control (RBAC) before creating workspaces and projects. To
configure RBAC, visit :ref:`rbac`.

.. attention::

   RBAC is only available on Determined Enterprise Edition.

Create Workspaces and Projects
==============================

Determined lets you organize and control access to your experiments by team or department. To do
this, you can create :ref:`workspaces` based on your RBAC groups.

Configure Checkpoint Storage
============================

To configure checkpoint storage, visit :ref:`checkpoint-storage`.

.. toctree::
   :hidden:

   Overview <self>
   PostgreSQL <postgresql>
   Set Up Clients <setup-clients>
   Internet Access <internet-access>
   Firewall Rules (Port Reference) <firewall-rules>
