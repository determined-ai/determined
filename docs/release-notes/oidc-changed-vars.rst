:orphan:

**Breaking Change**

-  Authentication: In the enterprise edition, in the master configuration, the
   ``oidc.groups_claim_name`` setting that is used to set the string value of the authenticator's
   claim name for groups has been changed to ``oidc.groups_attribute_name``. Similarly, the
   ``oidc.display_name_claim_name`` setting that is used to set the user's display name in
   Determined has been changed to ``oidc.display_name_attribute_name``.
