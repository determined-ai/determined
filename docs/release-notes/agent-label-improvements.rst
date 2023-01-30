:orphan:

**Bug Fixes**

-  Agent: Fix a bug where if a job was submitted with ``resources.agent_label`` and dynamic agents
   were configured, agents would be provisioned until the max number of instances were reached while
   the job remained in queue.

**Improvements**

-  Agent: A warning will be given if a job is submitted with ``resources.agent_label`` with a label
   that no agents connected have.
