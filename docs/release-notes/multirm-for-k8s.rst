:orphan:

**New Features**

-  Kubernetes: Add ability to set up the Determined master service on one Kubernetes cluster and
   manage workloads across different Kubernetes clusters. Additional non-default resource managers
   and resource pools are configured under the master configuration options
   ``additional_resource_managers`` and ``resource_pools`` (additional resource managers are
   required to have at least one resource pool defined). Additional resource managers and their
   resource pools must have unique names. For more information, visit :ref:master configuration
   <master-config-reference>. Support for notebooks and other workloads that require proxying is
   currently under development.

-  WebUI: Add ability to view resource manager name for resource pools.

-  API/CLI/WebUI: Route any requests to resource pools not defined in the master configuration to
   the default resource manager, not any additional resource manager, if defined.

-  Configuration: Add a ``name`` and ``metadata`` field to resource manager section in the master
   configuration. Add an ``additional_resource_managers`` section that follows the
   ``resource_manager`` and ``resource_pool`` configuration pattern.
