.. _auto-posix:

##############################
 Automatic POSIX User Linking
##############################

Determined supports automatic POSIX user linking based on OIDC/SAML claims. This feature streamlines
user management by automatically associating SSO identities with POSIX users on your cluster.

***************
 Configuration
***************

To enable automatic POSIX user linking, you must configure your OIDC or SAML integration to include
the necessary claims. The exact configuration depends on your identity provider.

OIDC Configuration
==================

For OIDC, add the following to your master configuration:

.. code:: yaml

   oidc:
     auto_provision_users: true
     posix_user_claim: "preferred_username"  # or the appropriate claim for your setup

SAML Configuration
==================

For SAML, add the following to your master configuration:

.. code:: yaml

   saml:
     auto_provision_users: true
     posix_user_attribute: "uid"  # or the appropriate attribute for your setup

*******
 Usage
*******

Once configured, when a user authenticates via SSO, Determined will automatically:

#. Check for the specified claim/attribute in the SSO response.
#. If found, use this value to link the SSO identity to a POSIX user on the cluster.
#. If the POSIX user doesn't exist, create it (if your configuration allows).

This process happens transparently to the end-user, providing a seamless SSO experience while
maintaining proper POSIX permissions on your cluster.

*************************
 Security Considerations
*************************

-  Ensure that your SSO provider is correctly configured to provide the necessary claims/attributes.
-  Regularly audit your user mappings to ensure they remain accurate and up-to-date.
-  Consider implementing additional access controls or monitoring for sensitive operations.

By leveraging automatic POSIX user linking, you can simplify user management, enhance security, and
provide a smoother experience for your users.
