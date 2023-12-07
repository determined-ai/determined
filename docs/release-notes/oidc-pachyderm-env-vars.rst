:orphan:

**New Features**

-  Authentication: Users can now provide a pachyderm address in the master config under
   integrations.pachyderm.address. This address will be added as an environment variable called
   ``PACHD_ADDRESS`` in the task container. The OIDC raw ID token will also be available as an
   environment variable called ``DEX_TOKEN`` in the task container.
