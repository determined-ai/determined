.. _oauth:

#########################
 OAuth 2.0 Configuration
#########################

.. attention::

   OAuth 2.0 applies only to Determined Enterprise Edition.

Only the SCIM endpoints are supported.

Determined EE allows requests to certain endpoints to be authenticated using OAuth 2.0 with the
authorization code flow.

**********************
 Enable OAuth Support
**********************

To enable OAuth support, set ``scim.auth.type`` to ``oauth`` in the Determined :ref:`master
configuration <master-config-reference>`.

The values you'll need to configure an OAuth client application are as follows:

-  The authorization endpoint, which is the hostname of the Determined master followed by
   ``/oauth2/authorize``.

-  The access token endpoint, which is the hostname of the Determined master followed by
   ``/oauth2/token``.

-  The client ID and secret, which are obtained using the Determined CLI:

   .. code::

      det oauth client add <descriptive client name> <domain of redirect URI>
      # For example:
      det oauth client add okta https://system-admin.okta.com

   The output of that command will look like the following:

   .. code::

      Client ID:     5d9bb6c1b423215f7eb0d719fffb39dda2d0d864252389da5061615d8da6887a
      Client secret: 37e96a2a27e20004477dbdc60c2143ee984817bc6b3a0016182a2fc15707b9c2

   .. warning::

      There is no other way to obtain the secret. Make sure not to lose it before configuring your
      client.

********************
 List OAuth Clients
********************

Use the CLI to listing OAuth clients:

.. code::

   det oauth client list

**********************
 Remove OAuth Clients
**********************

Use the CLI to remove OAuth clients:

.. code::

   det oauth client remove <client ID>
