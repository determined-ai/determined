.. _access-tokens:

###############
 Access Tokens
###############

Access tokens provide a secure way to authenticate automated workflows without requiring frequent user login. These tokens can be created, managed, and revoked as needed, enhancing both security and convenience for your workflows.

Creating Access Tokens
======================

To create a new access token, use the following CLI command:

.. code::

   det token create [username] --expiration-days DAYS --description DESCRIPTION

For example:

.. code::

   det token create determined --expiration-days 30 --description "Automated testing token"

This command will output the token ID and the actual token. Make sure to save the token securely, as it won't be displayed again.

Managing Access Tokens
======================

Access Token Permissions
------------------------

Determined includes a RBAC role called ``TokenCreator``. This role allows users to create, view, and revoke their own access tokens. The ``TokenCreator`` role can only be assigned globally.

Users with the ``TokenCreator`` role can perform the following actions:

- Create access tokens for themselves
- View their own active and revoked tokens
- Revoke their own tokens

Administrators and users with appropriate permissions can manage tokens for all users.

List Tokens
-----------

To view all active access tokens:

.. code::

   det token list

You can also use options to display revoked tokens.

Describe Tokens
---------------

To get detailed information about specific tokens:

.. code::

   det token describe TOKEN_ID [TOKEN_ID ...]

Edit Tokens
-----------

To update a token's description:

.. code::

   det token edit TOKEN_ID --description "New description"

Revoking Tokens
---------------

To revoke an access token:

.. code::

   det token revoke TOKEN_ID

Using Access Tokens
===================

To authenticate using an access token:

.. code::

   det token login YOUR_ACCESS_TOKEN

This will create a session authenticated with the token's associated user.

API Endpoints
=============

You can also use the following API endpoints to manage access tokens:

- ``POST /api/v1/tokens``: Create a new access token
- ``GET /api/v1/tokens``: Retrieve a list of access tokens
- ``PATCH /api/v1/tokens/{token_id}``: Edit an existing access token

For detailed API usage, please refer to our API documentation.

Security Considerations
=======================

- Treat access tokens like passwords. Never share them or commit them to version control.
- Define an appropriate lifespan for your tokens based on your use case.
- Regularly audit and rotate your access tokens.
- Revoke tokens immediately if they are no longer needed or may have been compromised.

Access tokens enhance automation while maintaining strong security protocols by allowing tighter control over token usage and expiration.
