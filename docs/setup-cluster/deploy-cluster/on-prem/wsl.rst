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

-  WSL 2 `installed and enabled <https://learn.microsoft.com/en-us/windows/wsl/install>`_ on your
   Windows machine.

-  An Ubuntu or Red Hat Enterprise Linux-based WSL distribution installed from the Microsoft Store,
   such as:

      -  `Ubuntu 22.04 LTS <https://www.microsoft.com/store/productId/9PDXGNCFSCZV>`_
      -  `AlmaLinux 9 <https://www.microsoft.com/store/productId/9P5RWLM70SN9>`_
      -  `Oracle Linux 9 <https://www.microsoft.com/store/productId/9MXQ65HLMC27>`_
      -  `Pengwin Enterprise <https://www.microsoft.com/store/productId/9P70GX2HQNHN>`_

-  `systemd enabled <https://learn.microsoft.com/en-us/windows/wsl/wsl-config#systemd-support>`_
   within your chosen WSL distribution.

**Recommended**

-  Windows 10 version 1903, or later. Recommended: Windows 11 version 22H2.
-  `Windows Terminal <https://www.microsoft.com/store/productId/9N0DX20HK701>`_.

.. _enable_systemd:

Enable ``systemd``
==================

Edit the configuration file to enable ``systemd`` within your WSL distribution. To do this:

-  Open a terminal window in your WSL distribution.

-  Add ``systemd=true`` to the ``[boot]`` section of ``/etc/wsl.conf`` in your WSL distribution:

   .. code::

      echo '[boot]' >> /etc/wsl.conf && echo 'systemd=true' >> /etc/wsl.conf

-  Then restart your WSL distribution:

   .. code::

      wsl --shutdown <distribution name>

.. _wsl_installation_using_packages:

**************************************
 Install using Debian or RPM packages
**************************************

This section provides instructions for installing Determined on WSL using Debian or RPM packages.

.. _packages_postgresql:

Install PostgreSQL using ``apt`` or ``yum``
===========================================

Since Determined uses a PostgreSQL database to store experiment and trial metadata, start by
installing PostgreSQL. You'll need PostgreSQL 10 or later.

   **Debian Distributions**

   On Debian distributions, use the following command:

   .. code::

      sudo apt install postgresql

   **Red Hat Distributions**

   On Red Hat distributions, you'll need to configure the PostgreSQL yum repository as described in
   the `Red Hat Linux documentation <https://www.postgresql.org/download/linux/redhat>`_. Then,
   install version 10:

   .. code::

      sudo yum install postgresql-server -y
      sudo postgresql-setup initdb
      sudo systemctl start postgresql.service
      sudo systemctl enable postgresql.service

The authentication methods enabled by default may vary depending on the provider of your PostgreSQL
distribution. To enable the ``determined-master`` to connect to the database, ensure that an
appropriate authentication method is configured in the ``pg_hba.conf`` file.

When configuring the database connection in :ref:`configure_the_cluster`, note the following:

-  If you specify the ``db.hostname`` property, you must use a PostgreSQL ``host`` (TCP/IP)
   connection.
-  If you omit the ``db.hostname`` property, you must use a PostgreSQL ``local`` (Unix-domain
   socket) connection.

Finally, create a database for Determined's use and configure a system account that Determined will
use to connect to the database.

For example, executing the following commands will create a database named ``determined``, create a
user named ``determined`` with the password ``determined-password``, and grant the user access to
the database:

.. code::

   sudo -u postgres psql
   postgres=# CREATE DATABASE determined;
   postgres=# CREATE USER determined WITH ENCRYPTED PASSWORD 'determined-password';
   postgres=# GRANT ALL PRIVILEGES ON DATABASE determined TO determined;

.. _packages_determined:

Install the Determined Master and Agent
=======================================

To find the latest release of Determined, visit the `Determined repo
<https://github.com/determined-ai/determined/releases/latest>`_.

Download the appropriate Debian or RPM package file, which will have the name
``determined-master_VERSION_linux_amd64.[deb|rpm]`` (where ``VERSION`` is the actual version, e.g.,
|version|). Similarly, the agent package is named
``determined-agent_VERSION_linux_amd64.[deb|rpm]``.

Install the master and agent package on one machine.

   **Debian Distributions**

   On Debian distributions, use the following command:

   .. code::

      sudo apt install <path to downloaded package>

   **Red Hat Distributions**

   On Red Hat distributions, use the following command:

   .. code::

      sudo rpm -i <path to downloaded package>

Before running the Determined agent, :ref:`install Docker <install-docker>` on each agent machine.

.. note::

   If you are not using Docker Desktop, you may disregard the prompt to use Docker Desktop and allow
   Docker to be installed within the WSL distribution.

.. _packages_configure_the_cluster:

Configure and Start the Cluster
===============================

Ensure that an instance of PostgreSQL is running and accessible from the machine where the
Determined master will run.

To start the Determined master, you'll need to first edit the master and agent configuration files.

Edit the :ref:`YAML configuration files <topic-guides_yaml>` at ``/etc/determined/master.yaml`` (for
the master) and ``/etc/determined/agent.yaml`` (for the agent) as appropriate for your setup.

.. important::

   Ensure that the user, password, and database name correspond to your PostgreSQL configuration.

In ``/etc/determined/master.yaml``:

.. code::

   db:
     host: localhost
     port: <PostgreSQL port, e.g., 5432 by default>
     name: <Database name, e.g., determined>
     user: <PostgreSQL user, e.g., postgres>
     password: <Database password>

In ``/etc/determined/agent.yaml``:

.. code::

   master_host: localhost
   master_port: <Master port, e.g., 8080 by default>

Start the master by typing the following command:

.. code::

   sudo systemctl start determined-master

.. note::

   You can also run the master directly using the command ``determined-master``. This may be useful
   when experimenting with Determined such as when you want to quickly test different configuration
   options before writing them to the configuration file.

Optionally, you can configure the master to start upon launching the WSL distro by using the
following command:

.. code::

   sudo systemctl enable determined-master

Verify that the master started successfully by viewing the log.

.. code::

   journalctl -u determined-master

You should see logs indicating that the master can successfully connect to the database, and the
last line should indicate ``http server started`` on the configured WebUI port (8080 by default).
You can also validate that the WebUI is running by navigating to ``http://<master>:8080`` with your
web browser (or ``https://<master>:8443`` if TLS is enabled). You should see ``No Agents`` on the
right side of the top navigation bar.

Start the agent on each agent machine.

.. code::

   sudo systemctl start determined-agent

Similarly, the agent can be run with the command ``determined-agent``.

Optionally, you can configure the agent to start upon launching the WSL distro by using the
following command:

.. code::

   sudo systemctl enable determined-agent

Verify that each agent started successfully by viewing the log.

.. code::

   journalctl -u determined-agent

You should see logs indicating that the agent started successfully, detected compute devices, and
connected to the master. On the Determined WebUI, you should now see slots available, both on the
right-hand side of the top navigation bar and if you select the ``Cluster`` view in the left-hand
navigation panel.

Launch the Determined WebUI from within WSL.

.. code::

   powershell.exe /C start http://localhost:8080

The Determined WebUI opens in your browser.

.. _wsl_installation_using_docker_desktop:

******************************
 Install using Docker Desktop
******************************

This section provides instructions for installing Determined on WSL using Docker Desktop.

.. _docker_desktop:

Install Docker Desktop
======================

Install `Docker Desktop on Windows <https://www.docker.com/products/docker-desktop/>`_.

Ensure the Docker daemon is reachable from your WSL distribution.

   Open the ``Settings`` dialog from the Docker Desktop tray icon, and select ``Resources``. Under
   ``WSL Integration``, select ``Enable integration with my default WSL distro``, and enable
   integration for the WSL distribution where you will be working with Determined.

.. _docker_desktop_postgresql:

Pull the PostgreSQL Image
=========================

Pull the official Docker image for PostgreSQL. We recommend using the version listed below.

.. code::

   docker pull postgres:10

This image is not provided by Determined AI. For more information, visit its `Docker Hub page
<https://hub.docker.com/_/postgres>`_.

.. _docker_desktop_determined:

Pull the Determined AI Image
============================

Pull the Docker image for the master or agent on each machine where these services will run. There
is a single master container running in a Determined cluster and typically there is one agent
container running on a given machine. A single machine can host both the master container and an
agent container.

To run the commands below, replace ``VERSION`` with a valid Determined version, such as the current
version, |version|:

.. code::

   docker pull determinedai/determined-master:VERSION
   docker pull determinedai/determined-agent:VERSION

.. _docker_desktop_start_cluster:

Start the Cluster
=================

The cluster can now be started, first by starting the database, then by launching the Determined
master and agent containers.

.. _docker_desktop_start_postgresql:

Start the PostgreSQL Container
==============================

To start the PostgreSQL container, use the following command. Replace ``<DB password>`` with the
password you would like to use for the database:

.. code::

   docker run \
       --name determined-db \
       -p 5432:5432 \
       -v determined_db:/var/lib/postgresql/data \
       -e POSTGRES_DB=determined \
       -e POSTGRES_PASSWORD=<DB password> \
       postgres:10

.. _docker_desktop_get_wsl_ip:

Obtain the WSL IP Address
=========================

To allow Determined to reach the PostgreSQL container, you will need to determine the IP address.

Run the following command to determine the IP address of the WSL distribution and store it as an
environment variable:

.. code::

   export WSL_IP=$(hostname -I | awk '{print $1}')

.. _docker_desktop_start_determined_master:

Start the Determined Master
===========================

To start the master container, run the following command, replacing ``<DB password>`` with the
database password:

..
   code:

   docker run \
       --name determined-master \
       -p 8080:8080 \
       -e DET_DB_HOST=$WSL_IP \
       -e DET_DB_NAME=determined \
       -e DET_DB_PORT=5432 \
       -e DET_DB_USER=postgres \
       -e DET_DB_PASSWORD=<DB password> \
       determinedai/determined-master:VERSION

Optionally, you may now launch the Determined WebUI from within WSL:

.. code::

   powershell.exe /C start http://localhost:8080

.. _docker_desktop_start_determined_agent:

Start the Determined Agent
==========================

To start the agent container, run the following command:

.. code::

   docker run \
       -v /var/run/docker.sock:/var/run/docker.sock \
       --name determined-agent \
       -e DET_MASTER_HOST=$WSL_IP \
       -e DET_MASTER_PORT=8080 \
       determinedai/determined-agent:VERSION

Optionally, you may now launch the Determined WebUI from within WSL to verify the agent is running
and connected:

.. code::

   powershell.exe /c start http://$WSLIP:8080/det/clusters

Determined internally makes use of `Fluent Bit <https://fluentbit.io>`__. The agent uses the
``fluent/fluent-bit:1.9.3`` Docker image at runtime. It will attempt to pull the image
automatically. If the agent machines in the cluster are not able to connect to Docker Hub, you must
manually place the image onto the agent machines in the cluster before Determined can run. To
specify a different image to use for running Fluent Bit (generally to make use of a custom Docker
registry---the image should not normally need to be changed otherwise), use the agent's
``--fluent-logging-image`` command-line option or ``fluent_logging_image`` config file option.

To ensure proper GPU access for the agent container, use the ``--gpus`` flag to specify the GPUs.
Failure to include this flag will result in the agent not having access to any GPUs. For example:

.. code::

   # Use all GPUs.
   docker run --gpus all ...
   # Use any four GPUs (selected by Docker).
   docker run --gpus 4 ...
   # Use the GPUs with the given IDs or UUIDs.
   docker run --gpus '"device=1,3"' ...

You can also disable and enable GPUs at runtime using the ``det slot disable`` and ``det slot
enable`` CLI commands, respectively.

.. _docker_desktop_manage_cluster:

Manage the Cluster
==================

By default, ``docker run`` runs in the foreground. You can stop a container simply by pressing
**Control-C**. If you wish to keep Determined running for the long term, consider running the
containers `detached <https://docs.docker.com/engine/reference/run/#detached--d>`_ and/or with
`restart policies <https://docs.docker.com/config/containers/start-containers-automatically/>`_. You
can also use the :ref:`deployment tool <install-using-deploy>`.
