:orphan:

**Deprecations**

-  Singularity, Podman, and Apptainer Container runtimes for AgentRM: Launching a Singluarity/Podman/Apptainer container runtimes for Agent is no longer supported. Docker is the only option that is supported.

-  Determined Agent on Slurm/PBS: Slurmcluster with Determined Agents is not supported any more. For detailed instructions on existing ways to deploy, visit :ref:deploy-on-slurm-pbs. This change was announced in version 0.33.0.