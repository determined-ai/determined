.. _hpc-with-agent:

####################
 Agent on Slurm/PBS
####################

As an alternative to using the HPC Launcher, you may instead utilize the Determined agent. In this
usage model, the system administrator creates a custom resource pool for each Determined user. You
then start a Determined agent on one or more compute nodes of the cluster using Slurm or PBS
commands to provide the resources for your resource pool. As work is submitted to this resource
pool, it is distributed to the set of available agents. If your Slurm/PBS job is terminated (for
example due to a time limit) before your Determined work is completed, your Determined work remains
in your resource pool until additional agents are started. You may add additional resources to your
resource pool by starting additional agents on your cluster. If your Determined work is complete
before any time limits are hit on the Slurm/PBS job providing resources, you terminate the agent
jobs manually using Slurm/PBS commands.

The primary advantages of this model are:

#. You have dedicated access to the compute resources provided by the agents you start for the
   duration of your HPC job. This can provide more predictable throughput as it avoids contention in
   a highly utilized cluster.

#. Your Determined experiments are seen by the workload manager as a single large job, rather than
   many smaller jobs. In some HPC environments, larger jobs are given preference in workload manager
   scheduling.

#. If you have jobs of different sizes sharing the same set of resources, you reduce the potential
   for fragmentation where larger jobs may be delayed in running because the free resources are
   distributed across many nodes.

#. It eliminates the need for user impersonation, which the HPC Launcher uses to submit jobs to the
   Slurm or PBS workload manager on your behalf, using a sudo configuration.

There are several disadvantages to this model as well:

#. You must interact with Slurm or PBS directly to submit and terminate jobs. Using the HPC launcher
   provides a more seamless user experience that focuses solely on interacting with Determined
   commands and interfaces.

#. Overall system utilization will likely be less. Direct human control over resource allocation and
   release introduces inefficiency. If you fail to keep sufficient work queued up in your resource
   pool or fail to terminate the Determined agents when you are through, you prevent other users
   from accessing those resources.

*****************************************
 Install the Determined Master and Agent
*****************************************

Before users can make use of Determined agents, a system administrator must provide the following:

#. The system administrator installs the on-premise Determined master component as described in the
   :doc:`/setup-cluster/deploy-cluster/on-prem/linux-packages` document, and the Determined agent on
   all nodes of the cluster, but does not enable or start the ``determined-agent.service``.

#. The system administrator creates a custom resource pool in the :ref:`cluster-resource-pools`
   configuration for each Determined user in the ``master.yaml``. A fragment for creating custom
   resource pools for ``user`` and ``user2`` using the default settings is as follows:

   .. code:: yaml

      resource_pools:
        - pool_name: user1
        - pool_name: user2

   It is recommended that :ref:`rbac` be used to limit access to the intended user of each of these
   resource pools.

***************************************
 Create a per-user Agent Configuration
***************************************

This step may be completed either by the system administrator or the intended user. In a
cluster-wide shared directory (examples in this section use ``$HOME``), create an ``agent.yaml``
file. Below is a minimal example using a resource pool named for the user (``$USER``) and
``singularity`` as the container runtime platform. If configured using variables such as ``$HOME``,
a single ``agent.yaml`` could be shared by all users.

.. code:: yaml

   master_host: master.mycluster.com
   master_port: 8090
   resource_pool: $USER
   container_runtime: singularity

There are several other settings commonly configured in the `agent.yaml` which are listed in the
table below. For the full list of options, see :ref:`agent-config-reference`.

+----------------------------+----------------------------------------------------------------+
| Option                     | Description                                                    |
+============================+================================================================+
| ``image_root``             | To avoid multiple image downloads, configure an image cache as |
|                            | per :ref:`singularity-image-cache`                             |
+----------------------------+----------------------------------------------------------------+
| ``container_runtime``      | Instead of ``singularity``, you could specify ``podman`` as    |
|                            | the container runtime.                                         |
+----------------------------+----------------------------------------------------------------+
| ``security``               | Secure the communications between the master and agent using   |
|                            | TLS. Configure the sections of the ``security`` block as per   |
|                            | :ref:`tls`.                                                    |
+----------------------------+----------------------------------------------------------------+

****************************************************
 Start Per-User Agents to Provide Compute Resources
****************************************************

The user may then start one or more agents to provide resources to their resource pool using the
agent.yaml configured above.

In the command examples below, it is assumed that the agent.yaml for a given user is provided in
`$HOME``. Paths may need to be updated depending on your local configuration.

On Slurm, you can allocate resources with the ``srun`` or ``sbatch`` commands with the desired
resource configuration options.

.. code:: bash

   srun --gpus=8 /usr/bin/determined-agent  --config-file $HOME/agent.yaml

or

.. code:: bash

   sbatch -N4 --gpus-per-node=tesla:4  --wrap="srun /usr/bin/determined-agent  --config-file $HOME/agent.yaml"

On PBS, you can launch the agent on multiple nodes with the qsub command.

.. code:: bash

   qsub -l select=2:ngpus=4 -- /opt/pbs/bin/pbsdsh -- /usr/bin/determined-agent --config-file $HOME/agent.yaml

You can add incremental resources to your resource pool, by submitting an additional job and
starting additional agents.

**************************************************
 Launch Jobs and Experiments on the Resource Pool
**************************************************

You can then submit experiments or other tasks to the agents you have started by selecting the
proper resource pool. The resource pool to be used can be specified on the command line or via the
experiment config using the ``resources.resource_pool`` setting.

.. code:: bash

   det command run --config resources.resource_pool=$USER hostname

*******************************
 Release the Cluster Resources
*******************************

When your jobs and experiments have been completed, be sure to release the resources by canceling
your Slurm/PBS job.
