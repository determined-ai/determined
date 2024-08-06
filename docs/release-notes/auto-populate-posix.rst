:orphan:

**New Features**

-  Master Configuration: Add support for POSIX claims in the master configuration. It now accepts
   `agent_uid_attribute_name`, `agent_gid_attribute_name`, `agent_user_name_attribute_name`, or
   `agent_group_name_attribute_name`. Refer to the :ref:OIDC master configuration
   <master-config-oidc> or :ref:SAML master configuration <master-config-saml> for details. If any
   of these fields are configured, they will be used and synced to the database.
