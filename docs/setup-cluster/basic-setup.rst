:orphan:

.. _basic-setup:

#########################
 Set Up Determined (WIP)
#########################

.. meta::
   :description: These basic instructions help you get started with Determined by setting up your training environment.

To get started with Determined you'll need to set up your training environment. To set up your
training environment, follow the steps below.

.. note::

   Your training environment can be a local development machine, an on-premise GPU cluster, or cloud
   resources.

****************************
 Step 1 - Set Up PostgreSQL
****************************

The first step is to set up PostgreSQL on the environment of your choice.

For example, to set up PostgreSQL on Docker, visit Install Determined Using Docker:
:ref:`Preliminary Steps <install-postgres-docker>`.

.. note::

   Visit the :ref:`Cluster Deployment Guide by Environment <setup-checklists>` for a detailed
   checklist for each environment.

*********************************
 Step 2 - Install ``DET_MASTER``
*********************************

Install the Determined Master and Agent. The preferred method for installing the Agent is to use
Linux packages. The recommended alternative to Linux packages is Docker.

Install Using Linux Packages
============================

To install the ``DET_MASTER`` using Linux packages, visit :ref:`Install Determined Using Linux
Packages--Install the Determined Master and Agent <install-det-linux>`.

.. note::

   This method is required when using Slurm.

Install Using Docker Container (Or Equivalent)
==============================================

To install the ``DET_MASTER`` using Docker, visit :ref:`Install Determined Using Docker
<install-using-docker>`.

Install on Kubernetes
=====================

To install the ``DET_MASTER`` using Kubernetes, visit :ref:`Install Determined on Kubernetes
<install-on-kubernetes>`.

********************************
 Step 3 - Set Up TLS (Optional)
********************************

Agent
=====

To find out how to set up TLS for Agents, visit :ref:`Transport Security Layer--Agent Configuration
<tls-agent-config>`.

Kubernetes
==========

To set up TLS on Kubernetes, choose one of the following methods:

-  type here
-  type here

Slurm
=====

To set up TLS on Slurm:

********************************
 Step 4 - Set Up SSO (Optional)
********************************

.. attention::

   SSO is only supported on the Determined Enterprise Edition.

To set up SSO, follow these instructions:

-  x
-  x
-  x

Only changes with Kubernetes.

***********************************
 Step 5 - Set Up Compute Resources
***********************************

Linux Packages
==============

x

Docker
======

x

Slurm
=====

x

Kubernetes
==========

x

*********************************************
 Step 6 - Set Up Monitoring Tools (Optional)
*********************************************

The following monitoring tools are officially supported: Prometheus/Grafana

Prometheus
==========

x

Grafana
=======

x
