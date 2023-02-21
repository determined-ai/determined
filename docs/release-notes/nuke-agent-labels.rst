:orphan:

**Breaking Changes**

-  Cluster: ``resources.agent_label`` task option and agent config ``label`` option are no longer
   supported and will be ignored. If you are not explicitly using these options, or only use single
   empty or non-empty label value per resource pool, no changes are necessary. Otherwise, cluster
   admins should create a resource pool for each existing ``resource_pool`` + ``agent_label``
   combination, and reconfigure agents to use these new pools. Cluster users should update their
   tasks to use the new resource pool names.
