.. _install-using-linux-packages:

###################
 Via Linux Packages
###################

This user guide provides step-by-step instructions for installing Determined using Linux packages.

Determined releases Debian and RPM packages for installing the Determined master and agent as
systemd services on machines running Linux.

You have two options for installing the Determined master and agent:

-  Using Debian packages on Ubuntu 16.04 or 18.04, or
-  Using Red Hat 7-based Linux distributions (e.g., Red Hat Enterprise Linux, CentOS, Oracle Linux,
   and Scientific Linux).

*******************
 Preliminary Setup
*******************

PostgreSQL
==========

Determined uses a PostgreSQL database to store experiment and trial metadata. You may either use a
Docker container or your Linux distribution's package and service.

.. note::

   If you are using an existing PostgreSQL installation, we recommend confirming that
   ``max_connections`` is at least 96, which is sufficient for Determined.

Run PostgreSQL in Docker
------------------------

#. Pull the official Docker image for PostgreSQL. We recommend using version 10 or greater.

   .. code::

      docker pull postgres:10

   This image is not provided by Determined AI; please see `its Docker Hub page
   <https://hub.docker.com/_/postgres>`_ for more information.

#. Start PostgreSQL as follows:

   .. code::

      docker run \
          -d \
          --restart unless-stopped \
          --name determined-db \
          -p 5432:5432 \
          -v determined_db:/var/lib/postgresql/data \
          -e POSTGRES_DB=determined \
          -e POSTGRES_PASSWORD=<Database password> \
          postgres:10

   If the master will connect to PostgreSQL via Docker networking, exposing port 5432 via the ``-p``
   argument isn't necessary; however, you may still want to expose it for administrative or
   debugging purposes. In order to expose the port only on the master machine's loopback network
   interface, pass ``-p 127.0.0.1:5432:5432`` instead of ``-p 5432:5432``.

Install PostgreSQL using ``apt`` or ``yum``
-------------------------------------------

#. Install PostgreSQL 10 or greater.

   **Debian Distributions**

   On Debian distributions, use the following command:

   .. code::

      sudo apt install postgresql-10

   **Red Hat Distributions**

   On Red Hat distributions, you'll need to configure the PostgreSQL yum repository as described in
   the `Red Hat Linux documentation <https://www.postgresql.org/download/linux/redhat>`_. Then,
   install version 10:

   .. code::

      sudo yum install postgresql-server -y
      sudo postgresql-setup initdb
      sudo systemctl start postgresql.service
      sudo systemctl enable postgresql.service

#. The authentication methods enabled by default may vary depending on the provider of your
   PostgreSQL distribution. To enable the ``determined-master`` to connect to the database, ensure
   that an appropriate authentication method is configured in the ``pg_hba.conf`` file.

   When configuring the database connection as described in :ref:`configure_the_cluster`, note the
   following:

   -  If you specify the ``db.hostname`` property, you must use a PostgreSQL ``host`` (TCP/IP)
      connection.
   -  If you omit the ``db.hostname`` property, you must use a PostgreSQL ``local`` (Unix domain
      socket) connection.

#. Finally, create a database for Determined's use and configure a system account that Determined
   will use to connect to the database.

   For example, executing the following commands will create a database named ``determined``, create
   a user named ``determined`` with the password ``determined-password``, and grant the user access
   to the database:

   .. code::

      sudo -u postgres psql
      postgres=# CREATE DATABASE determined;
      postgres=# CREATE USER determined WITH ENCRYPTED PASSWORD 'determined-password';
      postgres=# GRANT ALL PRIVILEGES ON DATABASE determined TO determined;

Install the Determined Master and Agent
=======================================

#. Find the latest release of Determined by visiting the `Determined repo
   <https://github.com/determined-ai/determined/releases/latest>`_.

#. Download the appropriate Debian or RPM package file, which will have the name
   ``determined-master_VERSION_linux_amd64.[deb|rpm]`` (where ``VERSION`` is the actual version,
   e.g., |version|). Similarly, the agent package is named
   ``determined-agent_VERSION_linux_amd64.[deb|rpm]``.

#. Install the master package on one machine in your cluster, and the agent package on each agent
   machine.

   **Debian Distributions**

   On Debian distributions, use the following command:

      .. code::

         sudo apt install <path to downloaded package>

   **Red Hat Distributions**

   On Red Hat distributions, use the following command:

      .. code::

         sudo rpm -i <path to downloaded package>

   Before running the Determined agent, :ref:`install Docker <install-docker>` on each agent
   machine. If the machine has GPUs, ensure that the :ref:`NVIDIA Container Toolkit
   <validate-nvidia-container-toolkit>` is working as expected.

.. _configure_the_cluster:

*********************************
 Configure and Start the Cluster
*********************************

#. Ensure that an instance of PostgreSQL is running and accessible from the machine where the
   Determined master will run.

#. Edit the :ref:`YAML configuration files <topic-guides_yaml>` at ``/etc/determined/master.yaml``
   (for the master) and ``/etc/determined/agent.yaml`` (for each agent) as appropriate for your
   setup.

   .. important::

      Ensure that the user, password, and database name correspond to your PostgreSQL configuration.

   .. code::

      db:
        host: <PostgreSQL server IP or hostname, e.g., 127.0.0.1 if running on the master>
        port: <PostgreSQL port, e.g., 5432 by default>
        name: <Database name, e.g., determined>
        user: <PostgreSQL user, e.g., postgres>
        password: <Database password>

#. Start the master by typing the following command:

   .. code::

      sudo systemctl start determined-master

   .. note::

      You can also run the master directly using the command ``determined-master``. This may be
      useful when experimenting with Determined, such as when you want to quickly test different
      configuration options before writing them to the configuration file.

#. Optionally, configure the master to start on boot.

   .. code::

      sudo systemctl enable determined-master

#. Verify that the master started successfully by viewing the log.

   .. code::

      journalctl -u determined-master

   You should see logs indicating that the master can successfully connect to the database, and the
   last line should indicate ``http server started`` on the configured WebUI port (8080 by default).
   You can also validate that the WebUI is running by navigating to ``http://<master>:8080`` with
   your web browser (or ``https://<master>:8443`` if TLS is enabled). You should see ``No Agents``
   on the right-hand side of the top navigation bar.

#. Start the agent on each agent machine.

   .. code::

      sudo systemctl start determined-agent

   Similarly, the agent can be run with the command ``determined-agent``.

#. Optionally, configure the agent to start on boot.

   .. code::

      sudo systemctl enable determined-agent

#. Verify that each agent started successfully by viewing the log.

   .. code::

      journalctl -u determined-agent

   You should see logs indicating that the agent started successfully, detected compute devices, and
   connected to the master. On the Determined WebUI, you should now see slots available, both on the
   right-hand side of the top navigation bar, and if you select the ``Cluster`` view in the
   left-hand navigation panel.

.. _socket-activation:

*******************
 Socket Activation
*******************

The master can be configured to use `systemd socket activation
<https://0pointer.de/blog/projects/socket-activation.html>`__, allowing it to be started
automatically on demand (e.g., when a client makes a network connection to the port) and restarted
with reduced loss of connection state. To switch to socket activation, run the following commands:

.. code::

   sudo systemctl disable --now determined-master
   sudo systemctl enable --now determined-master.socket

When socket activation is in use, the port on which the master listens is configured differently;
the port listed in the master config file is not used, since systemd manages the listening socket.
The default socket unit for Determined is configured to listen on port 8080. To use a different
port, run:

.. code::

   sudo systemctl edit determined-master.socket

which will open a text editor window. To change the listening port, insert the following text (with
the port number substituted appropriately) into the editor and then exit the editor:

.. code::

   [Socket]
   ListenStream=
   ListenStream=0.0.0.0:<port>

For example, you might want to configure the master to listen on port 80 for HTTP traffic or on port
443 if using :ref:`TLS <tls>`.

After updating the configuration, run the following commands to put the change into effect (this
will restart the master):

.. code::

   sudo systemctl stop determined-master
   sudo systemctl restart determined-master.socket

See the systemd documentation on `socket unit files
<https://www.freedesktop.org/software/systemd/man/systemd.socket.html>`__ or `systemctl
<https://www.freedesktop.org/software/systemd/man/systemctl.html>`__ for more information.

********************
 Manage the Cluster
********************

To configure a service to start running automatically when its machine boots up, run ``sudo
systemctl enable <service>``, where the service is ``determined-master`` or ``determined-agent``.
You can also use ``sudo systemctl enable --now <service>`` to enable and immediately start a service
in one command.

To view the logging output of a service, run ``journalctl -u <service>``.

To manually stop a service, run ``sudo systemctl stop <service>``.
