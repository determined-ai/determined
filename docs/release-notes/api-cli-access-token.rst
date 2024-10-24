:orphan:

**New Features**

-  API/CLI: Add support for access tokens. Add the ability create and administer access tokens for
   users to authenticate in automated workflows. Users can define the lifespan of these tokens,
   making it easier to securely authenticate and run processes. Users can set global defaults and
   limits for the validity of access tokens by configuring ``default_lifespan_days`` and
   ``max_lifespan_days`` in the master configuration. Setting ``max_lifespan_days`` to ``-1``
   indicates an **infinite** lifespan for the access token. This feature enhances automation while
   maintaining strong security protocols by allowing tighter control over token usage and
   expiration.

   -  CLI:

      -  ``det token create``: Create a new access token.
      -  ``det token login``: Sign in with an access token.
      -  ``det token edit``: Update an access token's description.
      -  ``det token list``: List all active access tokens, with options for displaying revoked
         tokens.
      -  ``det token describe``: Show details of specific access tokens.
      -  ``det token revoke``: Revoke an access token.

   -  API:

      -  ``POST /api/v1/tokens``: Create a new access token.
      -  ``GET /api/v1/tokens``: Retrieve a list of access tokens.
      -  ``PATCH /api/v1/tokens/{token_id}``: Edit an existing access token.
