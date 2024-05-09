:orphan:

**Deprecation Notice**

The following features may be removed in future releases, in order to provide a more stable user experience and simplify onboarding. The goal is for Agent Resource Manager environments will work out-of-the-box with minimal custom configuration.

- Agent Resource Manager:
  - Container Runtimes: Limit support to Docker for Agent Resource Managers.
  - Job Scheduling: Moving a job will require updating its priority; a job cannot be moved within the same priority group. Round-robin and fair share schedulers are no longer supported; the priority scheduler is recommended.  Most scheduling needs can be provided with the priority scheduler; focusing on one feature-rich scheduler eliminates noise in setup and configuration.
  - AMD GPUs: Nvidia GPUs remain supported. 

- Machine Architectures: PPC64/POWER builds are no longer supported. This applies to all environments.