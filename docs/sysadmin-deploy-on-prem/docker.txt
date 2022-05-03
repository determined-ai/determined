.. _install-using-docker:

#################################
 Install Determined Using Docker
#################################

*******************
 Preliminary Setup
*******************

#. :ref:`Install Docker <install-docker>` on all machines in the cluster. If the agent machines have
   GPUs, ensure that the :ref:`Nvidia Container Toolkit <validate-nvidia-container-toolkit>` on each
   one is working as expected.

#. Pull the official Docker image for PostgreSQL. We recommend using the version listed below.

   .. code::

      docker pull postgres:10

   This image is not provided by Determined AI; please see `its Docker Hub page
   <https://hub.docker.com/_/postgres>`_ for more information.

#. Pull the Docker image for the master or agent on each machine where these services will run.
   There is a single master container running in a :ref:`Determined cluster
   <det-system-architecture>`, and typically there is one agent container running on a given
   machine. A single machine can host both the master container and an agent container. Run the
   commands below, replacing ``VERSION`` with a valid Determined version, such as the current
   version, |version|:

   .. code::

      docker pull determinedai/determined-master:VERSION
      docker pull determinedai/determined-agent:VERSION

**************************************
 Configuring and Starting the Cluster
**************************************

PostgreSQL
==========

The following command starts the PostgreSQL container:

.. code::

   docker run \
       --name determined-db \
       -p 5432:5432 \
       -v determined_db:/var/lib/postgresql/data \
       -e POSTGRES_DB=determined \
       -e POSTGRES_PASSWORD=<DB password> \
       postgres:10

In order to expose the port only on the master machine's loopback network interface, pass ``-p
127.0.0.1:5432:5432`` instead of ``-p 5432:5432``. If you choose to run in host networking mode,
pass ``--network host`` instead of ``-p 5432:5432``.

Determined Master
=================

Determined master :ref:`configuration values <master-configuration>` can come from a file,
environment variables, or command-line arguments.

To start the master with a configuration file, we recommend starting from our `default master
configuration file
<https://raw.githubusercontent.com/determined-ai/determined/master/master/packaging/master.yaml>`_,
which contains a listing of the available options and descriptions for them. Download and edit the
``master.yaml`` configuration file as appropriate and start the master container with the edited
configuration:

.. code::

   docker run \
       -v "$PWD"/master.yaml:/etc/determined/master.yaml \
       determinedai/determined-master:VERSION

To start the master with environment variables instead of a configuration file:

.. code::

   docker run \
       --name determined-master \
       -p 8080:8080 \
       -e DET_DB_HOST=<PostgreSQL hostname or IP> \
       -e DET_DB_NAME=determined \
       -e DET_DB_PORT=5432 \
       -e DET_DB_USER=postgres \
       -e DET_DB_PASSWORD=<DB password> \
       determinedai/determined-master:VERSION

Regarding the hostname for PostgreSQL, if the ``determined-db`` container and the
``determined-master`` container are running on the same machine, you may find it easier to run both
of them with host networking (``--network host``) and use ``127.0.0.1`` here.

In order to prevent the master from listening on port 8080 on all network interfaces on the master
machine, you may specify the loopback interface in the published port mapping, i.e., ``-p
127.0.0.1:8080:8080``.

Determined Agents
=================

As is the case for the master, Determined agent :ref:`configuration values <agent-configuration>`
can come from a file, environment variables, or command-line arguments.

To start the agent with a configuration file, we recommend starting from our `default agent
configuration file
<https://raw.githubusercontent.com/determined-ai/determined/master/agent/packaging/agent.yaml>`_,
which contains a listing of the available options and descriptions for them. Download and edit the
``agent.yaml`` configuration file as appropriate and start the agent container with the edited
configuration:

.. code::

   docker run \
       -v /var/run/docker.sock:/var/run/docker.sock \
       -v "$PWD"/agent.yaml:/etc/determined/agent.yaml \
       determinedai/determined-agent:VERSION

Note that the agent container must bind mount the host's Docker daemon socket. This allows the agent
container to orchestrate the containers that execute trials and other tasks.

If you are providing command-line arguments to the container (e.g., using ``--master-port`` as
opposed to the ``DET_MASTER_PORT`` environment variable), ``run`` must be provided as the first
argument:

.. code::

   docker run \
       -v /var/run/docker.sock:/var/run/docker.sock \
       -v "$PWD"/agent.yaml:/etc/determined/agent.yaml \
       determinedai/determined-agent:VERSION \
       run --master-port=8080

To start an agent container with environment variables instead of a configuration file:

.. code::

   docker run \
       -v /var/run/docker.sock:/var/run/docker.sock \
       --name determined-agent \
       -e DET_MASTER_HOST=<Determined master hostname or IP> \
       -e DET_MASTER_PORT=8080 \
       determinedai/determined-agent:VERSION

When agents are on different machines than the master, the Determined master hostname or IP should
just be how you expect your agent machines to reach the master machine.

However, if the agents are on the same machine as the master, ``127.0.0.1`` will typically not work,
unless both the master and agent containers were started with ``--network host``. Note that even
that in that case, you will also have to configure workload containers to use ``host`` network mode,
as described :ref:`below <network-host>`. Otherwise, if the master machine has a static IP address
from your router, you can use that. What is important is that it is reliably addressable from both
inside and outside of docker containers (since the Fluentbit container will always use host
networking).

Determined internally makes use of `Fluent Bit <https://fluentbit.io>`__. The agent uses the
``fluent/fluent-bit:1.9.3`` Docker image at runtime. It will attempt to pull the image
automatically; if the agent machines in the cluster are not able to connect to Docker Hub, the image
must be manually placed on them before Determined can run. In order to specify a different image to
use for running Fluent Bit (generally to make use of a custom Docker registry---the image should not
normally need to be changed otherwise), use the agent's ``--fluent-logging-image`` command-line
option or ``fluent_logging_image`` config file option.

Selecting GPUs
--------------

The ``--gpus`` flag should be used to specify which GPUs the agent container will have access to;
without it, the agent will not have access to any GPUs. For example:

.. code::

   # Use all GPUs.
   docker run --gpus all ...
   # Use any four GPUs (selected by Docker).
   docker run --gpus 4 ...
   # Use the GPUs with the given IDs or UUIDs.
   docker run --gpus '"device=1,3"' ...

GPUs can also be disabled and enabled at runtime using the ``det slot disable`` and ``det slot
enable`` CLI commands, respectively.

.. _network-host:

Docker Networking for Master, Agents, and Workloads
===================================================

As with any Docker container, the networking mode of the master and agent containers can be changed
using the ``--network`` option to ``docker run``. In particular, host mode networking (``--network
host``) can be useful to optimize performance and in situations where a container needs to handle a
large range of ports, as it does not require network address translation (NAT) and no
"userland-proxy" is created for each port.

.. note::

   if you want to run workload containers in host networking mode, you will have to configure the
   ``task_container_defaults`` in the :ref:`master.yaml <cluster-configuration>`; the ``--network``
   argument to master or agent containers will not affect how the workload containers are lauched.

The host networking driver only works on Linux hosts, and is not supported on Docker Desktop for
Mac, Docker Desktop for Windows, or Docker EE for Windows Server.

See `Docker's documentation <https://docs.docker.com/network/host/>`_ for more details.

Note that at this time, even if you run the agents in a named docker network (e.g. ``--network
my-named-network``), the workloads launched by the agent will execute in a different docker network.
This will affect address resolution if you try to set the master hostname as the master's container
name, as the workload containers will not be in the correct docker network to reach the master by
that name.

**********************
 Managing the Cluster
**********************

By default, ``docker run`` will run in the foreground, so that a container can be stopped simply by
pressing Control-C. If you wish to keep Determined running for the long term, consider running the
containers `detached <https://docs.docker.com/engine/reference/run/#detached--d>`_ and/or with
`restart policies <https://docs.docker.com/config/containers/start-containers-automatically/>`_.
Using :ref:`our deployment tool <install-using-deploy>` is also an option.
