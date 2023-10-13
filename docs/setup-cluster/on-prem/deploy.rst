.. _install-using-deploy:

#########################################
 Install Determined Using ``det deploy``
#########################################

This user guide provides instructions for using the ``det deploy`` command-line tool to deploy
Determined locally or in a production cluster. ``det deploy`` automates the process of starting
Determined as a collection of Docker containers.

You can also use ``det deploy`` to install Determined on the cloud. For more information, see the
:ref:`AWS <install-aws>` and :ref:`GCP <install-gcp>` installation guides.

In a typical production setup, the master and agent nodes run on separate machines. The master and
agent nodes can also run on a single machine, which is useful for local development. This user guide
provides instructions for both scenarios.

*******************
 Preliminary Setup
*******************

.. note::

   To use ``det deploy`` for local installations, Docker must be installed. For Docker installation
   instructions, visit :ref:`installation <install-docker>`.

Install the ``determined`` Python package by running

.. code::

   pip install determined

.. include:: ../../_shared/note-pip-install-determined.txt

.. _configuring-cluster-install:

*********************************
 Configure and Start the Cluster
*********************************

A configuration file is needed to set important values in the master, such as where to save model
checkpoints. For information about how to create a configuration file, see
:ref:`cluster-configuration`. There are also sample configuration files available.

.. note::

   ``det deploy`` will use a default configuration file if you don't provide one. It also
   transparently manages PostgreSQL along with the master, so the configuration options related to
   those services do not need to be set.

Deploy a Single-Node Cluster
============================

For local development or small clusters (such as a GPU workstation), you may wish to install both a
master and an agent on the same node. To do this, run one of the following commands:

.. code::

   # If the machine has GPUs:
   det deploy local cluster-up

   # If the machine doesn't have GPUs:
   det deploy local cluster-up --no-gpu

This will start a master and an agent on that machine. To verify that the master is running,
navigate to ``http://<master-hostname>:8080`` in a browser, which should bring up the Determined
WebUI. If you're using your local machine, for example, navigate to ``http://localhost:8080``.

In the WebUI, go to the ``Cluster`` page. You should now see slots available (either CPU or GPU,
depending on what hardware is available on the machine).

For single-agent clusters launched with:

.. code::

   det deploy local cluster-up --auto-work-dir <absolute directory path>

the cluster will automatically make the specified directory available to tasks on the cluster as
``./shared_fs``. If ``--auto-work-dir`` is not specified, the cluster will default to mounting your
home directory. This will allow you to access your local preferences and any relevant files stored
in the specified directory with the cluster's notebooks, shells, and tensorboard tasks. To disable
this feature, use:

.. code::

   det deploy local cluster-up --no-auto-work-dir

For production deployments, you'll want to :ref:`use a cluster configuration file
<configuring-cluster-install>`. To provide this configuration file to ``det deploy``, use:

.. code::

   det deploy local cluster-up --master-config-path <path to master.yaml>

Stop a Single-Node Cluster
==========================

To stop a Determined cluster, on the machine where a Determined cluster is currently running, run

.. code::

   det deploy local cluster-down

.. note::

   ``det deploy local cluster-down`` will not remove any agents created with ``det deploy local
   agent-up``. To remove these agents, use ``det deploy local agent-down``.

Deploy a Standalone Master
==========================

In many cases, your Determined cluster will consist of multiple nodes. In this case you will need to
start a master and agents separately. In order to start a standalone master, run:

.. code::

   det deploy local master-up

.. note::

   For production deployments, you'll want to :ref:`use a cluster configuration file.
   <configuring-cluster-install>` To provide this configuration file to ``det deploy``, use the flag
   ``--master-config-path <path to master.yaml>``.

To stop a running master, run:

.. code::

   det deploy local master-down

Deploy Agents
=============

To deploy a standalone agent on a machine, run one of the following commands:

.. code::

   # If the machine has GPUs:
   det deploy local agent-up <master_hostname>

   # If the machine doesn't have GPUs:
   det deploy local agent-up --no-gpu <master_hostname>

This will create an agent on that machine. To verify whether it has successfully connected to the
master, navigate to the WebUI and check whether slots have appeared on the ``Cluster`` page.

To launch the agent into a specific resource pool, use the ``--agent-resource-pool`` flag:

.. code::

   det deploy local agent-up --agent-resource-pool=<resource_pool> <master_hostname>

For more information about resource pools, see :ref:`resource-pools`.

To stop a running agent, run:

.. code::

   det deploy local agent-down
