.. _hpc-security-considerations:

######################################
 HPC Launcher Security Considerations
######################################

User authentication in Determined is enforced using :ref:`users`. Determined sends requests on
behalf of those authenticated users to the HPC launcher which then interacts with the underlying
workload manager to submit and control jobs via the user agent configured for the user account. A
Determined administrator must configure a Determined user's agent to enable them to launch Slurm/PBS
jobs.

Several security issues should be considered when deploying the launcher:

#. The specified ``resource_manager.user_name`` should be a unique, non-privileged user with
   authorization to interact with the deployed workload manager (Slurm/PBS). The launcher executes
   as a service using the configured ``resource_manager.user_name`` and
   ``resource_manager.group_name`` as specified in the :ref:`slurm/pbs section
   <cluster-configuration-slurm>` of the cluster configuration. The launcher can also be run as the
   ``root`` user but with the corresponding reduction in security isolation.

#. The launcher installs the necessary ``sudoers(5)`` configuration with the file
   ``/etc/sudoers.d/zz_launcher`` to enable the specified ``resource_manager.user_name`` to perform
   the following actions:

   -  Change the ownership of a directory tree to another user (from ``resource_manager.user_name``
      to the Determined user before the job starts, and from the Determined user back to the
      ``resource_manager.user_name`` after completion).

   -  Enable the execution Slurm/PBS commands on behalf of the Determined user to submit and control
      their jobs.

#. The set of users ``sudo`` authorizes for Slurm/PBS launch is controlled by
   ``resource_manager.sudo_authorized``. The default value is ``ALL``. The configuration of users
   always includes the ``!root`` to prevent privilege elevation.

#. The launcher installs the necessary ``sudoers(5)`` configuration to enable all users to generate
   a token for read-only interaction with the launcher REST API. This capability is intended for use
   when other components integrate with the launcher.

.. _sudo_configuration:

***********************
 Configuration of sudo
***********************

The ``sudo`` configuration necessary to enable the launcher to perform Slurm/PBS job management on
behalf of the requesting Determined user is automatically generated and applied during the startup
of the launcher service as specified in the :ref:`slurm/pbs section <cluster-configuration-slurm>`
of the cluster configuration. Configuration is added to the ``sudo`` configuration by the file
``/etc/sudoers.d/zz_launcher``. The configuration is dervied from the following values:

-  The authorized user is configured as ``resource_manager.user_name`` (shown below as
   ``launcher``).

-  The run-as user list is configured to authorize ``resource_manager.sudo_authorized`` (shown below
   as the default value of ``ALL``).

   -  A comma-separated list of user/group specifications identifying users for which the launcher
      can submit/control Slurm/PBS jobs using ``sudo``.
   -  The specification ``!root`` is automatically appended to this list to prevent privilege
      elevation.
   -  This may be a list of users or groups with exclusions (e.g. ``%slurmusers,localadmin,!guest``
      ).
   -  See the ``sudoers(5)`` definition of ``Runas_List`` for the full syntax of this value.

-  For Slurm, the authorized commands are the full path to each of the commands ``sacct``,
   ``salloc``, ``sbatch``, ``scancel``, ``scontrol``, ``sinfo``, ``squeue``, ``srun``.

-  For PBS, the authorized commands are the full path to ``qsub``, ``qstat``, ``qdel``,
   ``pbsnodes``.

The content of a typical ``/etc/sudoers.d/zz_launcher`` generated for Slurm is shown below:

.. code::

   launcher ALL= (root) NOPASSWD: /bin/chown -R * *
   launcher ALL= (root) NOPASSWD: /usr/bin/chown -R * *
   ALL ALL = (root) NOPASSWD: /opt/launcher/bin/user-keytool
   launcher ALL= (ALL, !root) NOPASSWD:SETENV: /usr/bin/sacct
   launcher ALL= (ALL, !root) NOPASSWD:SETENV: /usr/bin/salloc
   launcher ALL= (ALL, !root) NOPASSWD:SETENV: /usr/bin/sbatch
   launcher ALL= (ALL, !root) NOPASSWD:SETENV: /usr/bin/scancel
   launcher ALL= (ALL, !root) NOPASSWD:SETENV: /usr/bin/scontrol
   launcher ALL= (ALL, !root) NOPASSWD:SETENV: /usr/bin/sinfo
   launcher ALL= (ALL, !root) NOPASSWD:SETENV: /usr/bin/squeue
   launcher ALL= (ALL, !root) NOPASSWD:SETENV: /usr/bin/srun

As noted above, this file is regenerated during the startup of the launcher service. It should not
be edited directly and should be configured using the attributes provided in the :ref:`slurm/pbs
section <cluster-configuration-slurm>` of the cluster configuration.
