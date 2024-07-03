:orphan:

**Deprecations**

-  AgentRM: Launching a Singluarity/Podman/Apptainer container runtimes for Agent is no longer supported. Docker is the
   only option that is supported. This change only affects when container_runtime is set to podman, using a podman emulation layer is unchanged. If you want to use singularity, podman, or apptainer the Determined master enterprise edition still supports it. For
   detailed instructions on existing ways to deploy, visit :ref:deploy-on-slurm-pbs. This change was
   announced in version 0.33.0.
