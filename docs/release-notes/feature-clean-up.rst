:orphan:

**Deprecations**

To enhance stability and streamline the onboarding process, we may remove the following features in
future releases. Our goal is for Agent Resource Manager environments to function seamlessly
out-of-the-box with minimal customization required.

Agent Resource Manager:

-  Container Runtimes: Due to limited usage, we will limit supported container runtimes to Docker
   for the Agent Resource Manager. This does not impact Kubernetes, Slurm or PBS environments.

-  Job Scheduling: The default scheduler is now ``priority``. Support for round-robin and fair share
   schedulers has been discontinued. We recommend using the priority scheduler, as it meets most
   scheduling needs and simplifies configuration. To move a job, you will need to adjust its
   priority; jobs cannot be shifted within the same priority group.

-  AMD GPUs: Due to limited usage, we will limit supported accelerators to NVIDIA GPUs. If you have
   a use case requiring AMD GPU support with the Agent Resource Manager, please reach out to us via
   a `GitHub Issue <https://github.com/determined-ai/determined/issues>`__ or `community slack
   <https://join.slack.com/t/determined-community/shared_invite/zt-1f4hj60z5-JMHb~wSr2xksLZVBN61g_Q>`__!
   This does not impact Kubernetes or Slurm environments.

Machine Architectures: PPC64/POWER builds across all environments are no longer supported.
