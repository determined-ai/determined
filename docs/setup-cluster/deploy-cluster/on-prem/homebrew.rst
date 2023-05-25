.. _install-using-homebrew:

###########################################
 Install Determined Using Homebrew (macOS)
###########################################

This user guide provides step-by-step instructions for installing Determined using Homebrew.

Determined publishes a Homebrew tap for installing the Determined master and agent as Homebrew
services on macOS, for both Apple silicon and Intel hardware.

While it is most common to install the master and agent on the same machine, it is also possible to
install the master and agent on separate nodes, or install agents on multiple machines and connect
them to one master.

.. note::

   Due to the limitations of Docker networking on macOS, distributed training across multiple macOS
   agents is not supported.

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

#. Start the PostgreSQL server, and set up a database and default user.

   .. code::

      brew services start postgresql@14
      createuser postgres
      createdb determined

#. Start the Determined master service.

   .. code::

      brew services start determined-master

#. If needed, you can configure the master by editing ``/usr/local/etc/determined/master.yaml`` and
   restarting the service.

**********************
 Installation - Agent
**********************

#. The Determined agent uses Docker to run your workloads. For more information, visit :ref:`Docker
   for Mac installation instructions <install-docker-on-macos>`.

#. By default, Determined will store checkpoints in ``$(brew --prefix)/var/determined/data``, which
   is typically ``/usr/local/var/determined/data`` or ``/opt/homebrew/var/determined/data``. Make
   sure to configure it as a shared path for Docker for Mac in Docker -> Preferences... -> Resources
   -> File Sharing.

#. When installing on a different machine than the master, add Homebrew tap.

   .. code::

      brew tap determined-ai/determined

#. Install ``determined-agent`` package.

   .. code::

      brew install determined-agent

#. When installing on a different machine than the master, edit
   ``/usr/local/etc/determined/agent.yaml`` and change ``master_host`` and ``container_master_host``
   to your master network hostname, and ``master_port`` to your master network port.

#. Start the ``determined-agent`` service.

   .. code::

      brew services start determined-agent
