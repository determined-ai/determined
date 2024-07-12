:orphan:

**Breaking Changes**

-  Cluster: ``resources.agent_label`` task option and agent config ``label`` option have been
   removed. Beginning with 0.20.0 release, these options have been ignored. Please remove any
   remaining references from configuration files and use ``resource_pool`` instead.
