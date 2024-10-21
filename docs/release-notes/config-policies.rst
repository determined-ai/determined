:orphan:

**New Features**

- Cluster: Using config policies, administrators can now set limits on how users can define workloads (e.g.,
  experiments, notebooks, tensorboards, shells, and commands). Admins may define two types of configurations:
  - **Invariant Configs for Experiments**: Settings applied to all experiments within a scope (global or workspace). 
    Invariant configs for other tasks (e.g. notebooks, tensorboards, shells, and commands) is not yet supported.
  - **Constraints**: Restrictions that prevent users from exceeding resource limits within a scope. Constraints can 
    be set independently for experiments and tasks.

Visit :ref:`Config Policies <config-policies>` for more details. 
