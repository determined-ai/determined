:orphan:

**Improvement**

-  Silences RBAC audit logging for CanViewSensitiveAgent information, reducing flooding of logs by
   automated polling of GetAgents.
-  Obfuscate Slot ID's in agent summaries when user does not have permission to access sensitive
   agent information. This limits the /api/v1/agents to sharing Slot ID's only with ClusterAdmins.
