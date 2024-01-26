:orphan:

**New Features**

-  Experiments: Add ``resources.is_single_node`` option which disallows scheduling the trial across
   multiple nodes or pods, and forces it to be scheduled within a single container. If the requested
   ``slots_per_trial`` count is impossible to fulfill in the cluster, the experiment submission will
   be rejected.

**Improvements**

-  Notebooks, Shells, and Commands: On static agent-based clusters (not using dynamic cloud
   provisioning), when a ``slots`` request for a notebook, shell, or command cannot be fulfilled,
   it'll be rejected.
