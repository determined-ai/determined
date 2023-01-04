.. _install-using-homebrew:

###########################################
 Install Determined Using Homebrew (macOS)
###########################################

Determined publishes a homebrew tap for installing the Determined master and agent as homebrew
services on macOS, including both Apple Silicon and Intel hardware.

Most commonly, master and agent are installed on the same machine, but it's also possible to install
them on separate nodes, or install the agents on multiple machines and connect them to one master.

.. note::

   Due to the limitations of docker networking on macOS, distributed training across multiple macOS
   agents is currently not supported.

***********************
 Installation - Master
***********************

#. Add Homebrew tap.

   .. code::

      brew tap determined-ai/determined

#. Install ``determined-master`` package. Determined uses a PostgreSQL database to store metadata,
   and ``postgresql@14`` will be pulled in as a dependency.

   .. code::

      brew install determined-master

#. Start PostgreSQL server, setup a database and default user.

   .. code::

      brew services start postgresql@14
      createuser postgres
      createdb determined

#. Start Determined master service.

   .. code::

      brew services start determined-master

#. If needed, master can be configured by editing ``/usr/local/etc/determined/master.yaml`` and
   restarting the service.

**********************
 Installation - Agent
**********************

#. Determined agent uses docker to run your workloads. See Docker for Mac installation instructions
   :ref:`here <install-docker-on-macos>`.

#. When installing on a different machine than the master, add Homebrew tap.

   .. code::

      brew tap determined-ai/determined

#. Install ``determined-agent`` package.

   .. code::

      brew install determined-agent

#. When installing on a different machine than the master, edit
   ``/usr/local/etc/determined/agent.yaml`` and change ``master_host`` and ``container_master_host``
   to your master network hostname, and ``master_port`` to your master network port.

#. Start ``determined-agent`` service.

   .. code::

      brew services start determined-agent
