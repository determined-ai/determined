.. _upgrade-on-hpc:

#################################
 Upgrade Determined on Slurm/PBS
#################################

This procedure describes how to upgrade Determined on an HPC cluster managed by the Slurm or PBS
workload managers. Use this procedure when an earlier version of Determined is installed,
configured, and functioning properly.

#. Review the latest :ref:`slurm-requirements` and ensure all dependencies have been met.

#. Upgrade the launcher.

   For an example RPM-based installation, run:

   .. code:: bash

      sudo rpm -Uv hpe-hpc-launcher-<version>.rpm

   On Debian distributions, run:

   .. code:: bash

      sudo apt install ./hpe-hpc-launcher-<version>.deb

   The upgrade automatically updates and restarts the ``systemd`` ``launcher`` service.

#. Upgrade the on-premises Determined master component (not including the Determined agent) as
   described in the :ref:`upgrades` document.

   The upgrade does not automatically update the Determine master service. Reload the ``systemd``
   configuration and restart the Determine master service with the following commands.

   .. code:: bash

      sudo systemctl daemon-reload
      sudo systemctl restart determined-master.service
