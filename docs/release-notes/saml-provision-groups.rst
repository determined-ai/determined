:orphan:

**New Features**

-  Authentication: SAML users can be auto-provisioned upon their first login. To configure, set the
   ``saml.auto_provision_users`` option to True. If SCIM is enabled as well,
   ``auto_provision_users`` must be False.

-  Authentication: In the enterprise edition, add synchronization of SAML user group memberships
   with existing groups and SAML user display name with the Determined user display name. Configure
   by setting ``saml.groups_attribute_name`` to the string value of the authenticator's attribute
   name for groups and ``saml.display_name_attribute_name`` with the authenticator's attribute name
   for display name.

**Improvement**

-  Security: In the enterprise edition, expand the SAML user group memberships feature to provision
   groups upon each login. This can be done by setting ``saml.groups_attribute_name`` to the string
   value of the authenticator's attribute name for groups. Prior releases only matched group
   memberships between the authenticator and local Determined user groups, meaning that, if not
   found, local groups would not be created.
