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

To Do include https://docs.determined.ai/latest/setup-cluster/basic.html#master

The next step is to decide if you want to deploy the Determined Master on premises or on cloud.

.. tabs::

   .. tab::

      On Prem

      .. tabs::

         .. tab::

            Docker (Agent-Based)

            If the Determined Agent is your compute resource, you'll install the Determined Agent
            along with the Determined Master. The preferred method for installing the Agent is to
            use Linux packages. The recommended alternative to Linux packages is Docker.

            To install the Determined Master and Agent on premises, you'll first need to meet the
            installation requirements:

            -  :ref:`Installation Requirements <requirements>`

            Once you've met the installation requirements, select one of the following options:

            -  :ref:`Install Determined Using Docker <install-using-docker>`

         .. tab::

            Linux (Agent-Based)

            If the Determined Agent is your compute resource, you'll install the Determined Agent
            along with the Determined Master. The preferred method for installing the Agent is to
            use Linux packages. The recommended alternative to Linux packages is Docker.

            To install the Determined Master and Agent on premises, you'll first need to meet the
            installation requirements:

            -  :ref:`Installation Requirements <requirements>`

            Once you've met the installation requirements, select one of the following options:

            -  :ref:`Install Determined Using Linux Packages <install-using-linux-packages>`

         .. tab::

            Kubernetes

            To install the Determined Master on premises with Kubernetes, follow the steps below:

            -  :ref:`Deploy on Kubernetes <determined-on-kubernetes>`
            -  :ref:`Install Determined on Kubernetes <install-on-kubernetes>`

         .. tab::

            Slurm

            To install the Determined Master on premises with Slurm, follow the steps below:

            -  :ref:`sysadmin-deploy-on-hpc`

   .. tab::

      On Cloud

      .. tabs::

         .. tab::

            Agent-Based

            To install the Determined Master and Agent on cloud, select one of the following
            options:

            -  :ref:`AWS <install-aws>`
            -  :ref:`GCP <install-gcp>`

            .. note::

               When using AWS or GCP, ``det CLI`` manages the installation of the Determined Agent
               for you.

         .. tab::

            Kubernetes

            To install the Determined Master on cloud using Kubernetes, start here:

            -  :ref:`Install on Kubernetes <install-on-kubernetes>`

            After completing the step above, select one of the following options:

            -  :ref:`setup-eks-cluster`
            -  :ref:`setup-gke-cluster`
            -  :ref:`setup-aks-cluster`

To do include Firewall rules

The firewall rules must satisfy the following network access requirements for the master and agents.

Firewall Rules
==============

-  Inbound TCP to the master's network port from the Determined agent instances, as well as all
   machines where developers want to use the Determined CLI or WebUI. The default port is ``8443``
   if TLS is enabled and ``8080`` if not.

-  Outbound TCP to all ports on the Determined agents.

********************************
 Step 3 - Set Up TLS (Optional)
********************************

It is recommended to use :ref:`Transport Layer Security (TLS) <tls>`. However, if you do not require
the secure version of HTTP, you can skip this section.

-  Master-Only TLS

Add instructions.

-  Mutual TLS

Agent-Based

In an agent-based installation, Determined is the resource manager.

To set up TLS for Agents, visit :ref:`Transport Security Layer--Agent Configuration
<tls-agent-config>`.

-  Kubernetes TLS

Add instructions.

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

            To Do Kubernetes

            To find out how to set up SSO with Kubernetes, visit :ref:`tls-agent-config`. .. _saml:

         .. tab::

            To Do Other

            To set up SSO in any environment other than Kubernetes, visit :ref:`tls-agent-config`.

To validate Step 4, ensure the users can access the Determined cluster.

***********************************
 Step 5 - Set Up Compute Resources
***********************************

Step 5a - Set up Internet Access
================================

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

Step 5b - Firewall Rules (Port Reference) for Agents
====================================================

To do: some is correct, some incorrect, needs to be rewritten

The firewall rules must satisfy the following network access requirements for the master and agents.

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

*********************************************
 Step 6 - Set Up Monitoring Tools (Optional)
*********************************************

The following monitoring tools are officially supported: Prometheus/Grafana

.. tabs::

   .. tab::

      Prometheus

      Description and link to instructions.

   .. tab::

      Grafana

      Description and link to instructions.

.. _cluster-configuration:

*************************
 Configuring the Cluster
*************************

Common configuration reference: :doc:`/reference/deploy/config/common-config-options`

Master configuration reference: :doc:`/reference/deploy/config/master-config-reference`

Agent configuration reference: :doc:`/reference/deploy/config/agent-config-reference`

Basic Configuration
===================

To Do: Some of this is outdated (e.g., Docker Run)

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

To Do: Move to Commands and Shells

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

************
 Next Steps
************

RBAC
====

x

Workspaces
==========

x

Checkpoint Storage
==================

x

Deploying Your Cluster
======================

Once you have set up the necessary components for your chosen environment, you can configure the
environment. For detailed instructions by environment, visit the :ref:`Cluster Deployment Guide by
Environment <setup-checklists>`.

.. toctree::
   :hidden:

   Overview <self>
   PostgreSQL <postgresql>
   Set Up Clients <setup-clients>
