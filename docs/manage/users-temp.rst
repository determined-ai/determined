.. _users-temp:

##################################
 Managing Remote Users and Groups
##################################

This is a temporary page for the purpose of review. Upon approval, content will be merged into
:ref:`users`.

************************************************************
 Enabling Remote Authentication via Your IdP (Without SCIM)
************************************************************

You can auto manage your user provisioning and group management in Determined. Even if you do not
have SCIM enabled, you can still configure your Determined cluster to auto-provision users. With
this method, the user will be able to sign in to the Determined cluster using the email address they
use to authenticate via SSO.

.. note::

   This is only enabled for OIDC users.

To begin, you'll need set the following OIDC configuration parameter to ``true`` in your master
configuration file.

``auto_provision_users: true``

The system will also group sync via ``security.claim_group_assignments: match``.

The system will also set the username of the user to the same username the user uses to sign in to
their IdP (and it cannot be set independently).

For example here the OIDC claim is set to add the user to groups A and B. The display name is set as
well.

.. code::

   {
      groups: ['A', 'B'],
      displayName: "Cee Ray"
   }

These OIDC claims are sent in a Json Web Token from the IdP (e.g., Okta) by the admin.

This basically defines what info the IdP shares with the application (e.g., Determined) when the
user signs in.

For example:

.. code:: yaml

   oidc:
       enabled: true
       provider: "Okta"
       idp_recipient_url: "https://determined.example.com"
       idp_sso_url: "https://dev-00000000.okta.com"
       client_id: "xx0xx0"
       client_secret: "xx0xx0"
       auto_provision_users: true

Once this is configured, to sign in via SSO, the user simply enters their SSO-enabled email address.
For example to sign in to Determined with Okta, the user performs the following steps:

-  Visit the Determined URL, e.g., https://determined.example.com.
-  Under Sign in with Okta, enter your SSO email address.

Upon successfully signing in with their SSO email address, Determined authenticates the user and
adds the user to the user table.

Cluster admin administrators can view, manage, sort, and filter users via the user table in the
WebUI:

-  Sign in to the Determined cluster via the Determined URL, e.g., https://determined.example.com,
   as a cluster admin.
-  To view Admin Settings, select your profile in the upper left corner and then choose Admin.

You can add users, set existing users as remote users, edit users, perform bulk actions, etc.

***********************
 Managing Remote Users
***********************

You can allow Determined to manage remote users in your IdP in accordance with IAM best practices
and have that information passed to Determined upon each successful sign in via SSO without having
to manually modify the users or update them via SCIM.

You can also manually manage remote users and groups. One way to do this is via the WebUI. You can
also use the CLI.

Adding a New Remote User in the WebUI
=====================================

Steps

Editing an Existing User in the WebUI
=====================================

Steps
