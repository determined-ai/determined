:orphan:

.. _cluster-configuration:

########################
 Configuring the Cluster
########################

.. meta::
   :description: Follow these steps to set up a Determined training environment on-prem or on cloud.


After setting up your cluster using the advanced setup guide, you can configure it.

Cluster Configuration Resources
===============================

-  Common configuration reference: :doc:`/reference/deploy/config/common-config-options`
-  Master configuration reference: :doc:`/reference/deploy/config/master-config-reference`
-  Agent configuration reference: :doc:`/reference/deploy/config/agent-config-reference`

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
