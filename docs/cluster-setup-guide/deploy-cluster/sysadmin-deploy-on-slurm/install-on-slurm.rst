.. _install-on-slurm:

#############################
 Install Determined on Slurm
#############################

This document describes how to deploy Determined on a Slurm cluster.

The Determined master and launcher installation packages are configured for installation on a single
login or administrator Slurm cluster node.

***************************
 Install Determined Master
***************************

After the node has been selected and the
:doc:`/cluster-setup-guide/deploy-cluster/sysadmin-deploy-on-slurm/slurm-requirements` have been
fulfilled and configured, install and configure the Determined master:

#. Install the on-premises Determined master component as described in the
   :doc:`/cluster-setup-guide/deploy-cluster/sysadmin-deploy-on-prem/linux-packages` document.

#. Install the launcher.

   For an example RPM-based installation, run:

   .. code:: bash

      sudo rpm -ivh hpe-hpc-launcher-<version>.rpm

   The installation configures and enables the ``systemd`` ``launcher`` service, which provides
   Slurm management capabilities.

   If launcher dependencies are not satisfied, warning messages are displayed. Install or update
   missing dependencies or adjust the ``path`` in the next step to locate the dependencies.

.. _using_slurm:

*************************************************
 Configure and Verify Determined Master on Slurm
*************************************************

#. The launcher automatically adds a prototype ``resource_manager`` section for Slurm. Edit the
   provided ``resource_manager`` configuration section for your particular deployment. For RPM-based
   installations, the configuration file is typically the ``/etc/determined/master.yaml`` file.

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
          tres_supported: true

#. The installer provides default values, however, you should explicitly configure the following
   cluster options:

   +----------------------------+----------------------------------------------------------------+
   | Option                     | Experiment Type                                                |
   +============================+================================================================+
   | ``port``                   | Communication port used by the launcher. Update this value if  |
   |                            | there are conflicts with other services on your cluster.       |
   +----------------------------+----------------------------------------------------------------+
   | ``job_storage_root``       | Shared directory where job-related files are stored. This      |
   |                            | directory must be visible to the launcher and from the compute |
   |                            | nodes.                                                         |
   +----------------------------+----------------------------------------------------------------+
   | ``container_run_type``     | The container type to be launched on Slurm (``singularity`` or |
   |                            | ``podman``). The default type is ``singularity``.              |
   +----------------------------+----------------------------------------------------------------+
   | ``singularity_image_root`` | Shared directory where Singularity images are hosted. Unused   |
   |                            | unless ``container_run_type`` is ``singularity``. See          |
   |                            | :ref:`slurm-image-config` for details on how this option is    |
   |                            | used.                                                          |
   +----------------------------+----------------------------------------------------------------+
   | ``user_name`` and          | By default, the launcher runs from the root account. Create a  |
   | ``group_name``             | local account and group and update these values to enable      |
   |                            | running from another account.                                  |
   +----------------------------+----------------------------------------------------------------+
   | ``path``                   | If any of the launcher dependencies are not on the default     |
   |                            | path, you can override the default by updating this value.     |
   +----------------------------+----------------------------------------------------------------+

   See the :ref:`slurm section <cluster-configuration-slurm>` of the cluster configuration reference
   for the full list of configuration options.

   After changing values in the ``resource_manager`` section of the ``/etc/determined/master.yaml``
   file, restart the launcher service:

   .. code:: bash

      sudo systemctl restart launcher

#. Verify successful launcher startup using the ``systemctl status launcher`` command. If the
   launcher fails to start, check system log diagnostics, such as ``/var/log/messages`` or
   ``journalctl --since=yesterday -u launcher``, make the needed changes to the
   ``/etc/determined/master.yaml`` file, and restart the launcher.

   If the installer reported incorrect dependencies, verify that they have been resolved by changes
   to the ``path`` in the previous step:

   .. code:: bash

      sudo /etc/launcher/scripts/check-dependencies.sh

#. Reload the Determined master to get the updated configuration:

   .. code:: bash

      sudo systemctl restart determined-master

#. If the compute nodes of your cluster do not have internet connectivity to download Docker images,
   see :ref:`slurm-image-config`.

#. Verify the configuration by sanity-checking your Determined Slurm configuration:

   .. code:: bash

      det command run hostname

   A successful configuration reports the hostname of the compute node selected by Slurm to run the
   job.

#. Run a simple distributed training job such as the :doc:`/tutorials/pytorch-mnist-tutorial` to
   verify that it completes successfully. This validates Determined master and launcher
   communication, access to the shared filesystem, GPU scheduling, and highspeed interconnect
   configuration. For more complete validation, ensure that the ``slots_per_trial`` is at least
   twice the number of GPUs available on a single node.

*****************
 Configure Slurm
*****************

Determined should function with your existing Slurm configuration. The following steps are
recommended to optimize how Determined interacts with Slurm:

-  Enable Slurm for GPU Scheduling.

   Configure Slurm with `SelectType=select/cons_tres <https://slurm.schedmd.com/cons_res.html>`__.
   This enables Slurm to track GPU allocation instead of tracking only CPUs. If this is not
   available, you must change the :ref:`slurm section <cluster-configuration-slurm>`
   ``tres_supported`` option to ``false``.

-  Configure GPU Generic Resources (GRES).

   Determined works best when allocating GPUs. Information about what GPUs are available is
   available using GRES. You can use the `AutoDetect
   <https://slurm.schedmd.com/gres.html#AutoDetect>`__ feature to configure GPU GRES automatically.
   Otherwise, you should manually configure `GRES GPUs
   <https://slurm.schedmd.com/gres.html#GPU_Management>`__ such that Slurm can schedule nodes with
   the GPUs you want.

   For the automatic selection of nodes with GPUs, Slurm must be configured for ``GresTypes=gpu``
   and nodes with GPUs must have properly configured GRES indicating the presence of any GPUs. If
   Slurm GRES cannot be properly configured, specify the :ref:`slurm section
   <cluster-configuration-slurm>` ``gres_supported`` option to ``false``, and it is the user's
   responsibility to ensure that GPUs will be available on nodes selected for the job using other
   configurations such as targeting a specific resource pool with only GPU nodes, or specifying a
   Slurm constraint in the experiment configuration.

-  Ensure homogeneous Slurm partitions.

   Determined maps Slurm partitions to Determined resource pools. It is recommended that the nodes
   within a partition are homogeneous for Determined to effectively schedule GPU jobs.

   -  A Slurm partition with GPUs is identified as a CUDA resource pool.
   -  A Slurm partition with no GPUs is identified as an AUX resource pool.
   -  The Determined default resource pool is set to the Slurm default partition.

-  Tune the Slurm configuration for Determined job preemption.

   Slurm preempts jobs using signals. When a Determined job receives SIGTERM, it begins a checkpoint
   and graceful shutdown. To prevent unnecessary loss of work, it is recommended to set ``GraceTime
   (secs)`` high enough to permit the job to complete an entire Determined ``scheduling_unit``.

   To enable GPU job preemption, use ``PreemptMode=REQUEUE`` or ``PreemptMode=REQUEUE``, because
   ``PreemptMode=SUSPEND`` does not release GPUs so does not allow a higher-priority job to access
   the allocated GPU resources.
