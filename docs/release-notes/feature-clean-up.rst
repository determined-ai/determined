:orphan:

**Deprecations**

To enhance stability and streamline the onboarding process, we may remove the following features in
future releases. Our goal is for Agent Resource Manager environments to function seamlessly
out-of-the-box with minimal customization required.

Agent Resource Manager:

-  Container Runtimes: We will limit support to Docker for Agent Resource Managers. This does not
   impact Kubernetes or Slurm environments.

-  Job Scheduling: The default scheduler is now ``priority``. Support for round-robin and fair share
   schedulers has been discontinued. We recommend using the priority scheduler, as it meets most
   scheduling needs and simplifies configuration. To move a job, you will need to adjust its
   priority; jobs cannot be shifted within the same priority group.

-  AMD GPUs: Support will continue only for Nvidia GPUs. This applies only to agent resource
   mangers. It does not impact Kubernetes or Slurm environments.

Machine Architectures: PPC64/POWER builds across all environments are no longer supported.
