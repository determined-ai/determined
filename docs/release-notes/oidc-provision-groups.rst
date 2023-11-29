:orphan:

**Improvement**

Security: In the enterprise edition, expand the OIDC user group memberships feature to provision
groups upon each login. This can be done by setting `oidc.groups_claim_name` to the string value of
the authenticator's claim name for groups. Prior releases only matched group memberships between the
authenticator and local Determined user groups, meaning that, if not found, local groups would not
be created.
