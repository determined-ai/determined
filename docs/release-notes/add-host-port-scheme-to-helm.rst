:orphan:

**New Features**

-  Helm: Support configuring ``determined_master_host``, ``determined_master_port``, and
   ``determined_master_scheme``. These control how tasks address the Determined API server and are
   useful when installations span multiple Kubernetes clusters or there are proxies in between tasks
   and the master. Also, ``determined_master_host`` now defaults to the service host,
   ``<det_namespace>.<det_service_name>.svc.cluster.local``, instead of the service IP.
