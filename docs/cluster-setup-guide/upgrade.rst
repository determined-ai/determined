.. _upgrades:

.. _upgrades-troubleshootings:

#########
 Upgrade
#########

.. warning::

   There are occasionally incompatible changes introduced in new versions of Determined -- for
   example, the format of the :ref:`master and agent configuration files <cluster-configuration>`
   might change. While we try to preserve backward compatibility whenever possible, you should read
   the :ref:`release-notes` for a description of recent changes before upgrading Determined.

Upgrading an existing Determined installation requires the same steps as installing Determined for
the first time. Before starting an upgrade, first follow the steps below to safely shut down the
cluster. Once the upgrade is complete and Determined is restarted, all suspended experiments will be
resumed automatically.

#. Disable all Determined agents in the cluster:

   .. code::

      det -m <MASTER_ADDRESS> agent disable --all

   where ``MASTER_ADDRESS`` is the IP address or host name where the Determined master can be found.
   This will cause all tasks running on those agents to be checkpointed and terminated. The
   checkpoint process might take some time to complete; you can monitor which tasks are still
   running via ``det slot list``.

#. Take a backup of the Determined database using `pg_dump
   <https://www.postgresql.org/docs/10/app-pgdump.html>`_. This is a safety precaution in case any
   problems occur after upgrading Determined.

All users should also upgrade the CLI by running

.. code::

   pip install --upgrade determined
