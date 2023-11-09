:orphan:

**New Features**

-  Authentication: OIDC users can be auto-provisioned upon their first login. To configure, set the
   ``oidc.auto_provision_users`` option to True. If SCIM is enabled as well,
   ``auto_provision_users`` must be False.
