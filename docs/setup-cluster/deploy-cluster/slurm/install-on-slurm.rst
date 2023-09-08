.. _install-on-slurm:

#################################
 Install Determined on Slurm/PBS
#################################

This document describes how to deploy Determined on an HPC cluster managed by the Slurm or PBS
workload managers.

The Determined master and launcher installation packages are configured for installation on a single
login or administrator Slurm/PBS cluster node.

***************************
 Install Determined Master
***************************

After the node has been selected and the
:doc:`/setup-cluster/deploy-cluster/slurm/slurm-requirements` have been fulfilled and configured,
install and configure the Determined master:

#. Install the on-premises Determined master component (not including the Determined agent) as
   described in the :doc:`/setup-cluster/deploy-cluster/on-prem/linux-packages` document. Perform
   the installation and configuration steps, but stop before starting the ``determined-master``
   service, and continue with the steps below.

#. Install the launcher.

   For an RPM-based installation, run:

   .. code:: bash

      sudo rpm -ivh hpe-hpc-launcher-<version>.rpm

   On Debian distributions, instead run:

   .. code:: bash

      sudo apt install ./hpe-hpc-launcher-<version>.deb

   The installation configures and enables the ``systemd`` ``launcher`` service, which provides HPC
   management capabilities.

   If launcher dependencies are not satisfied, warning messages are displayed. Install or update
   missing dependencies or adjust the ``path`` and ``ld_library_path`` in the next step to locate
   the dependencies.

#. You may verify the installation integrity using the appropriate package manager command. See
   :ref:`hpc_package_verification`.

.. _using_slurm:

*******************************************************
 Configure and Verify Determined Master on HPC Cluster
*******************************************************

#. The launcher automatically adds a prototype ``resource_manager`` section for Slurm/PBS if not
   already present upon startup of the launcher service. Edit the provided ``resource_manager``
   configuration section for your particular deployment. For Linux package-based installations, the
   configuration file is typically the ``/etc/determined/master.yaml`` file.

   In this example, with Determined and the launcher colocated on a node named ``login``, the
   section might resemble:

   .. code:: yaml

      port: 8080
      ...
      resource_manager:
          type: slurm
          master_host: login
          master_port: 8080
          host: localhost
          port: 8181
          protocol: http
          container_run_type: singularity
          auth_file: /root/.launcher.token
          job_storage_root:
          path:
          ld_library_path:
          tres_supported: true
          slot_type: cuda

#. The installer provides default values, however, you should explicitly configure the following
   cluster options:

   +----------------------------+----------------------------------------------------------------+
   | Option                     | Experiment Type                                                |
   +============================+================================================================+
   | ``type``                   | The cluster workload manager (``slurm`` or ``pbs``).           |
   +----------------------------+----------------------------------------------------------------+
   | ``master_host``            | The host name of the Determined master. This is the name the   |
   |                            | compute nodes will utilize to communicate with the Determined  |
   |                            | master.                                                        |
   +----------------------------+----------------------------------------------------------------+
   | ``port``                   | Communication port used by the launcher. Update this value if  |
   |                            | there are conflicts with other services on your cluster.       |
   +----------------------------+----------------------------------------------------------------+
   | ``job_storage_root``       | Shared directory where job-related temporary files are stored. |
   |                            | The directory must be visible to the launcher and from the     |
   |                            | compute nodes. If ``user_name`` is configured as a user        |
   |                            | account other than ``root``, then the default value is         |
   |                            | ``$HOME/.launcher``.                                           |
   +----------------------------+----------------------------------------------------------------+
   | ``container_run_type``     | The container type to be launched on Slurm (``apptainer``,     |
   |                            | ``singularity``, ``enroot``, or ``podman``). The default is    |
   |                            | ``singularity``. Specify ``singularity`` when using Apptainer. |
   +----------------------------+----------------------------------------------------------------+
   | ``apptainer_image_root``   | Shared directory on all compute nodes where                    |
   | ``singularity_image_root`` | Apptainer/Singularity images are hosted. Unused unless         |
   |                            | ``container_run_type`` is ``singularity``. See                 |
   |                            | :ref:`slurm-image-config` for details on how this option is    |
   |                            | used.                                                          |
   +----------------------------+----------------------------------------------------------------+
   | ``user_name`` and          | By default, the launcher runs from the root account. Create a  |
   | ``group_name``             | local account and group and update these values to enable      |
   |                            | running from another account. This account must have access to |
   |                            | the Slurm/PBS command line to discover partitions and          |
   |                            | summarize cluster usage. See                                   |
   |                            | :ref:`hpc-security-considerations`.                            |
   +----------------------------+----------------------------------------------------------------+
   | ``path``                   | If any of the launcher dependencies are not on the default     |
   |                            | path, you can override the default by updating this value.     |
   +----------------------------+----------------------------------------------------------------+
   | ``gres_supported``         | Indicates that Slurm/PBS identifies available GPUs. The        |
   |                            | default is ``true``. See :ref:`slurm-config-requirements` or   |
   |                            | :ref:`pbs-config-requirements` for details.                    |
   +----------------------------+----------------------------------------------------------------+

   See the :ref:`slurm/pbs section <cluster-configuration-slurm>` of the cluster configuration
   reference for the full list of configuration options.

   After changing values in the ``resource_manager`` section of the ``/etc/determined/master.yaml``
   file, restart the launcher service:

   .. code:: bash

      sudo systemctl restart launcher

#. Verify successful launcher startup using the ``systemctl status launcher`` command. If the
   launcher fails to start, check system log diagnostics, such as ``/var/log/messages`` or
   ``journalctl --since="10 minutes ago" -u launcher``, make the needed changes to the
   ``/etc/determined/master.yaml`` file, and restart the launcher.

   If the installer reported incorrect dependencies, verify that they have been resolved by changes
   to the ``path`` and ``ld_library_path`` in the previous step:

   .. code:: bash

      sudo /etc/launcher/scripts/check-dependencies.sh

#. Reload the Determined master to get the updated configuration:

   .. code:: bash

      sudo systemctl restart determined-master

#. Verify successful determined-master startup using the ``systemctl status determined-master``
   command. If the launcher fails to start, check system log diagnostics, such as
   ``/var/log/messages`` or ``journalctl --since="10 minutes ago" -u determined-master``, make the
   needed changes to the ``/etc/determined/master.yaml`` file, and restart the determined-master.

#. If the compute nodes of your cluster do not have internet connectivity to download Docker images,
   see :ref:`slurm-image-config`.

#. If internet connectivity requires use of a proxy, make sure the proxy variables are defined as
   per :ref:`proxy-config-requirements`.

#. Log into Determined, see :ref:`users`. The Determined user must be linked to a user on the HPC
   cluster. If signed in with a Determined administrator account, the following example creates a
   Determined user account that is linked to the current user's Linux account.

   .. code:: bash

      det user create $USER
      det user link-with-agent-user --agent-uid $(id -u) --agent-gid $(id -g) --agent-user $USER --agent-group $(id -gn) $USER
      det user login $USER

   .. note::

      If an agent user has not been configured for a Determined username, jobs will run as user
      root. For more details see :ref:`run-as-user`.

#. Verify the configuration by sanity-checking your Determined configuration:

   .. code:: bash

      det command run hostname

   A successful configuration reports the hostname of the compute node selected by Slurm to run the
   job.

#. Run a simple distributed training job such as the :doc:`/tutorials/pytorch-mnist-tutorial` to
   verify that it completes successfully. This validates Determined master and launcher
   communication, access to the shared filesystem, GPU scheduling, and highspeed interconnect
   configuration. For more complete validation, ensure that the ``slots_per_trial`` is at least
   twice the number of GPUs available on a single node.
