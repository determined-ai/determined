:orphan:

**Deprecations**

-  AgentRM: As of version 0.33.0, support for Singularity, Podman, and Apptainer has been deprecated
   and is now officially removed. Docker is the only container runtime supported by Agent resource
   manager (AgentRM). However, you can still use Podman with AgentRM by utilizing the Podman
   emulation layer. For instructions, visit the Podman Desktop documentation and search for
   "Emulating Docker CLI with Podman". Additionally, you may need to configure
   ``checkpoint_storage`` in your experiment configuration or :ref:`master-config-reference`.

   In the enterprise edition, the Slurm Resource Manager continues to support Singularity, Podman,
   and Apptainer.
