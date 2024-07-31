.. _install-using-wsl:

################################################################
 Install Determined Using Windows Subsystem for Linux (Windows)
################################################################

This user guide provides step-by-step instructions for installing Determined on the Windows
Subsystem for Linux (WSL). You have two options for installation: using the Debian or RPM packages
provided by Determined, or using Docker containers published by Determined with Docker Desktop. In
this user guide, we'll focus on achieving a single-machine installation of Determined, with both the
master and agent running on the same machine within WSL.

.. _wsl_requirements:

**************
 Requirements
**************

**Minimum**

-  Windows 10 version 1903, or later.

-  Intel/AMD64 architecture device. Arm64 devices are not supported.

-  WSL 2 `installed and enabled <https://learn.microsoft.com/en-us/windows/wsl/install>`_ on your
   Windows machine.

-  An Ubuntu or an Enterprise Linux WSL distribution installed from the `Microsoft Store
   <https://apps.microsoft.com/home>`_, such as:

   -  Ubuntu 22.04 LTS
   -  AlmaLinux 9
   -  Oracle Linux 9
   -  Pengwin Enterprise

-  `systemd enabled <https://learn.microsoft.com/en-us/windows/wsl/wsl-config#systemd-support>`_
   within your chosen WSL distribution.

**Recommended**

-  Windows 11 version 22H2, or later.
-  `Windows Terminal <https://apps.microsoft.com/detail/9N0DX20HK701>`_.

.. _enable_systemd:

Enable ``systemd``
==================

Edit the configuration file to enable ``systemd`` within your WSL distribution. To do this:

-  Open a terminal window in your WSL distribution.

-  Add ``systemd=true`` to the ``[boot]`` section of ``/etc/wsl.conf`` in your WSL distribution:

   .. code::

      sudo bash -c "echo '[boot]' >> /etc/wsl.conf && echo 'systemd=true' >> /etc/wsl.conf"

-  Then shutdown WSL:

   .. code::

      wsl.exe --shutdown

-  Re-launch your WSL distribution.

#########################################
 Install Determined Using ``det deploy``
#########################################

This user guide provides instructions for using the ``det deploy`` command-line tool to deploy
Determined locally on WSL. ``det deploy`` automates the process of starting Determined as a
collection of Docker containers.

You can also use ``det deploy`` to install Determined on the cloud. For more information, see the
:ref:`AWS <install-aws>` and :ref:`GCP <install-gcp>` installation guides.

In a typical production setup, the master and agent nodes run on separate machines. The master and
agent nodes can also run on a single machine, which is useful for local development. This user guide
provides instructions for local development on WSL.

*******************
 Preliminary Setup
*******************

.. note::

   To use ``det deploy`` for local installations, Docker must be installed. For Docker installation
   instructions, visit :ref:`installation <install-docker>`.

Install pip if it is not already installed:

On Ubuntu:

.. code::

   sudo apt-get install python3-pip -y

On Enterprise Linux:

.. code::

   sudo dnf install python3-pip -y

Install the ``determined`` Python package by running:

.. code::

   pip install determined

.. include:: ../../../_shared/note-pip-install-determined.txt

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
navigate to ``http://localhost:8080`` in a browser, which should bring up the Determined WebUI.

To open the WebUI from WSL:

.. code::

   explorer.exe http://localhost:8080

The default username for the WebUI is ``determined`` and no password. After signing in, create a
:ref:`strong password <strong-password>`.

In the WebUI, go to the ``Cluster`` page. You should now see slots available (either CPU or GPU,
depending on what hardware is available on the machine).

For single-agent clusters launched with:

.. code::

   det deploy local cluster-up --auto-work-dir <absolute directory path>

the cluster will automatically make the specified directory available to tasks on the cluster as
``./shared_fs``. If ``--auto-work-dir`` is not specified, the cluster will default to mounting your
home directory. This will allow you to access your local preferences and any relevant files stored
in the specified directory with the cluster's notebooks, shells, and TensorBoard tasks. To disable
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
