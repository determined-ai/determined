######################
  Getting Started
######################

Two main components are required to use Determined:

-  the :ref:`Determined CLI <install-cli>`
-  a :ref:`Determined cluster <install-cluster>`.

Install the Determined CLI on your local development machine and install the Determined cluster in your training environment. Your training environement can be a local development machine, an on-premise GPU cluster, or cloud resources.

****************************
 Install the Determined CLI
****************************

The Determined CLI is a command line tool that lets you launch new experiments and interact
with a Determined cluster. To install the CLI, follow the :ref:`installation <install-cli>` instructions.

.. _install-cluster:

********************************
 Install the Determined cluster
********************************

A Determined cluster comprises a master and one or more agents . The cluster can be installed on Amazon Web Services (AWS), Google Cloud Platform (GCP), on-premise, or on a local development machine.

*****************
 Master Database
*****************

Each Determined cluster requires access to a `PostgreSQL <https://www.postgresql.org/>`_ database.
Additionally, Determined can use `Docker <https://www.docker.com/>`_ to run the master and agents.
Depending on your installation method, some of these services are installed for you:

-  On a cloud provider using ``det deploy``, Docker and PostgreSQL are preinstalled.
-  For on-premise using ``det deploy``, you need to install Docker.
-  For a manual installation, you need to install Docker and PostgreSQL.
