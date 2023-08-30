:orphan:

.. _installation-options:

#####################
 Installation Options
#####################

In this guide, you'll find basic and advanced cluster configuration options, as well as additional installation options include setting up clients, internet access, and firewall rules for the master and agent.

.. _cluster-configuration:

**********************
 Configure the Cluster
**********************

Common configuration reference: :doc:`/reference/deploy/config/common-config-options`

Master configuration reference: :doc:`/reference/deploy/config/master-config-reference`

Agent configuration reference: :doc:`/reference/deploy/config/agent-config-reference`

Basic Configuration
===================

The behavior of the master and agent can be controlled by setting configuration variables; this can
be done using a configuration file, environment variables, or command-line options. Although values
from different sources will be merged, we generally recommend sticking to a single source for each
service to keep things simple.

The master and the agent both accept an optional ``--config-file`` command-line option, which
specifies the path of the configuration file to use. Note that when running the master or agent
inside a container, you will need to make the configuration file accessible inside the container
(e.g., via a bind mount). For example, this command starts the agent using a configuration file:

.. code::

   docker run \
     -v `pwd`/agent-config.yaml:/etc/determined/agent-config.yaml \
     determinedai/determined-agent
     --config-file /etc/determined/agent-config.yaml

The ``agent-config.yaml`` file might contain

.. code:: yaml

   master_host: 127.0.0.1
   master_port: 8080

to configure the address of the Determined master that the agent will attempt to connect to.

Each option in the master or agent configuration file can also be specified as an environment
variable or a command-line option. To configure the behavior of the master or agent using
environment variables, specify an environment variable starting with ``DET_`` followed by the name
of the configuration variable. Underscores (``_``) should be used to indicate nested options: for
example, the ``logging.type`` master configuration option can be specified via an environment
variable named ``DET_LOGGING_TYPE``.

The equivalent of the agent configuration file shown above can be specified by setting two
environment variables, ``DET_MASTER_HOST`` and ``DET_MASTER_PORT``. When starting the agent as a
container, environment variables can be specified as part of ``docker run``:

.. code::

   docker run \
     -e DET_MASTER_HOST=127.0.0.1 \
     -e DET_MASTER_PORT=8080 \
     determinedai/determined-agent

The equivalent behavior can be achieved using command-line options:

.. code::

   determined-agent run --master-host=127.0.0.1 --master-port=8080

The same behavior applies to master configuration settings as well. For example, configuring the
host where the Postgres database is running can be done via a configuration file containing:

.. code:: yaml

   db:
     host: the-db-host

Equivalent behavior can be achieved by setting the ``DET_DB_HOST=the-db-host`` environment variable
or ``--db-host the-db-host`` command-line option.

In the rest of this document, we will refer to options using their names in the configuration file.
Periods (``.``) will be used to indicate nested options; for example, the option above would be
indicated by ``db.host``.

Advanced Configuration
======================

:ref:`Additional configuration settings <command-notebook-configuration>` for both commands and
shells can be set using the ``--config`` and ``--config-file`` options. Typical settings include:

-  ``bind_mounts``: Specifies directories to be bind-mounted into the container from the host
   machine. (Due to the structured values required for this setting, it needs to be specified in a
   config file.)

-  ``resources.slots``: Specifies the number of slots the container will have access to.
   (Distributed commands and shells are not supported; all slots will be on one machine and
   attempting to use more slots than are available on one machine will prevent the container from
   being scheduled.)

-  ``environment.image``: Specifies a custom Docker image to use for the container.

-  ``description``: Specifies a description for the command or shell to distinguish it from others.

.. _setup-clients:

***************
 Set Up Clients
***************

You can set up clients for interacting with the Determined Master through the Determined CLI. Follow
these instructions to set up clients.

Step 1 - Set ``DET_MASTER`` Environment Variable
================================================

Set the ``DET_MASTER`` environment variable, which is the network address of the Determined master.
You can override the value in the command line using the ``-m`` option.

.. note::

   You can skip this step when deploying locally.

Step 2 - Install the Determined CLI
===================================

The Determined CLI is a command-line tool that lets you launch new experiments and interact with a
Determined cluster. The CLI can be installed on any machine you want to use to access Determined. To
install the CLI, follow the :ref:`installation <install-cli>` instructions.

The ``-m`` or ``--master`` flag determines the network address of the Determined master that the CLI
connects to. If this flag is not specified, the value of the ``DET_MASTER`` environment variable is
used; if that environment variable is not set, the default address is ``localhost``. The master
address can be specified in three different formats:

-  ``example.org:port`` (if ``port`` is omitted, it defaults to ``8080``)
-  ``http://example.org:port`` (if ``port`` is omitted, it defaults to ``80``)
-  ``https://example.org:port`` (if ``port`` is omitted, it defaults to ``443``)

Examples:

.. code:: bash

   # Connect to localhost, port 8080.
   $ det experiment list

   # Connect to example.org, port 8888.
   $ det -m example.org:8888 e list

   # Connect to example.org, port 80.
   $ det -m http://example.org e list

   # Connect to example.org, port 443.
   $ det -m https://example.org e list

   # Connect to example.org, port 8080.
   $ det -m example.org e list

   # Set default Determined master address to example.org, port 8888.
   $ export DET_MASTER="example.org:8888"


.. _internet-access:

***********************
 Set Up Internet Access
***********************

-  The Determined Docker images are hosted on Docker Hub. Determined agents need access to Docker
   Hub for such tasks as building new images for user workloads.

-  If packages, data, or other resources needed by user workloads are hosted on the public Internet,
   Determined agents need to be able to access them. Note that agents can be :ref:`configured to use
   proxies <agent-network-proxy>` when accessing network resources.

-  For best performance, it is recommended that the Determined master and agents use the same
   physical network or VPC. When using VPCs on a public cloud provider, additional steps might need
   to be taken to ensure that instances in the VPC can access the Internet:

   -  On GCP, the instances need to have an external IP address, or a `GCP Cloud NAT
      <https://cloud.google.com/nat/docs/overview>`_ should be configured for the VPC.

   -  On AWS, the instances need to have a public IP address, and a `VPC Internet Gateway
      <https://docs.aws.amazon.com/vpc/latest/userguide/VPC_Internet_Gateway.html>`_ should be
      configured for the VPC.


.. _firewall-rules:

**********************
 Set Up Firewall Rules
**********************

The firewall rules must satisfy the following network access requirements for the master and agents.

Master
======

-  Inbound TCP to the master's network port from the Determined agent instances, as well as all
   machines where developers want to use the Determined CLI or WebUI. The default port is ``8443``
   if TLS is enabled and ``8080`` if not.

-  Outbound TCP to all ports on the Determined agents.

Agents
======

-  Inbound TCP from all ports on the master to all ports on the agent.

-  Outbound TCP from all ports on the agent to the master's network port.

-  Outbound TCP to the services that host the Docker images, packages, data, and other resources
   that need to be accessed by user workloads. For example, if your data is stored on Amazon S3,
   ensure the firewall rules allow access to this data.

-  Inbound and outbound TCP on all ports to and from each Determined agent. The details are as
   follows:

   -  Inbound and outbound TCP ports 1734 and 1750 are used for synchronization between trial
      containers.
   -  Inbound and outbound TCP port 12350 is used for internal SSH-based communication between trial
      containers.
   -  When using ``DeepSpeedTrial``, port 29500 is used by for rendezvous between trial containers.
   -  When using ``PyTorchTrial`` with the "torch" distributed training backend, port 29400 is used
      for rendezvous between trial containers
   -  For all other distributed training modes, inbound and outbound TCP port 12355 is used for GLOO
      rendezvous between trial containers.
   -  Inbound and outbound ephemeral TCP ports in the range 1024-65536 are used for communication
      between trials via GLOO.
   -  For every GPU on each agent machine, an inbound and outbound ephemeral TCP port in the range
      1024-65536 is used for communication between trials via NCCL.
   -  Two additional ephemeral TCP ports in the range 1024-65536 are used for additional intra-trial
      communication between trial containers.
   -  Each TensorBoard uses a port in the range 2600–2899
   -  Each notebook uses a port in the range 2900–3199
   -  Each shell uses a port in the range 3200–3599
