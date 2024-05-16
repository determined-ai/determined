:orphan:

**Deprecations**

To enhance stability and streamline the onboarding process, we may remove the following features in
future releases. Our goal is for Agent Resource Manager environments to function seamlessly
out-of-the-box with minimal customization required.

Agent Resource Manager:

-  Container Runtimes: We will limit support to Docker for Agent Resource Managers.

-  Job Scheduling: - Switching to `priority` as the default scheduler. Support for round-robin and
   fair share schedulers is discontinued. We recommend using the priority scheduler, as it meets
   most scheduling needs and simplifies configuration. - Moving a job will require adjusting its
   priority; jobs cannot be shifted within the same priority group.

-  AMD GPUs: Support will continue only for Nvidia GPUs.

Machine Architectures: PPC64/POWER builds across all environments are no longer supported.
