:orphan:

.. _setup-postgresql:

##################
 Set Up PostgreSQL
##################

Determined uses a PostgreSQL database to store experiment and trial metadata. 

.. note::

   If you are using an existing PostgreSQL installation, we recommend confirming that
   ``max_connections`` is at least 96, which is sufficient for Determined.


.. _install-postgres-docker:

temporary: replaces the Original section in docker.rst

*******************************************
 Setting Up a Docker PostgreSQL Environment
*******************************************

#. :ref:`Install Docker <install-docker>` on all machines in the cluster. If the agent machines have
   GPUs, ensure that the :ref:`NVIDIA Container Toolkit <validate-nvidia-container-toolkit>` on each
   one is working as expected.

#. Pull the official Docker image for PostgreSQL. We recommend using the version listed below.

   .. code::

      docker pull postgres:10

   This image is not provided by Determined AI; visit `its Docker Hub page
   <https://hub.docker.com/_/postgres>`_ for more information.

#. Pull the Docker image for the master or agent on each machine where these services will run.
   There is a single master container running in a Determined cluster, and typically there is one
   agent container running on a given machine. A single machine can host both the master container
   and an agent container. Run the commands below, replacing ``VERSION`` with a valid Determined
   version, such as the current version, |version|:

   .. code::

      docker pull determinedai/determined-master:VERSION
      docker pull determinedai/determined-agent:VERSION

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


.. _install-using-linux-packages-preliminary:

temporary: replaces the Original section in linux-packages.rst

*******************************************
 Setting Up a Linux PostgreSQL Environment
*******************************************

.. note::
   
   If you are installing Determined using Linux packages, you can use a Docker container or your Linux distribution's package and service.

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



*******************************************
 Next Steps
*******************************************

- :ref: `Set Up Determined Overview <basic-setup>`