:orphan:

**Deprecations**

-  AgentRM: Support for Singluarity, Podman, and Apptainer has been deprecated in 0.33.0 and is now
   removed. Docker is the only container runtime supported by Agent resource manager (AgentRM). It
   is still possible to use podman with AgentRM by using the podman emulation layer. For detailed
   instructions, follow steps in the link: `Emulating Docker CLI with Podman
   <https://podman-desktop.io/docs/migrating-from-docker/emulating-docker-cli-with-podman>`. You
   might need to also configure checkpoint_storage in experiment or master configurations: `Master
   Config Reference
   https://docs.determined.ai/latest/reference/deploy/master-config-reference.html#checkpoint-storage`

In enterprise edition, Slurm resource manager still supports singularity, podman, or apptainer use.
For detailed instructions, visit :ref:deploy-on-slurm-pbs.
