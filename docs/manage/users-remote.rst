.. _remote-users:

#######################
 Managing Remote Users
#######################

When using OIDC or SAML as your authentication protocol you can configure Determined to remote
provision and manage your users.

Remote user provisioning lets you include and synchronize any information about the user stored in
your IdP such as their username, display name, and groups. Once configured, you can manage remote
users without having to manually modify the users or update them via SCIM. Each time the remote user
accesses Determined, their information is synchronized.

.. include:: ../_shared/attn-enterprise-edition.txt

*******************************
 Enable Remote User Management
*******************************

Set the Auto Provision Option
=============================

The first step is to configure the :ref:`master configuration <master-config-reference>` file to
enable user auto-provisioning and the remote management of any information attached to the users.

.. tabs::

   .. tab::

      OIDC

      -  Enable OIDC.
      -  Set ``oidc.auto_provision_users`` option to ``true`` in your :ref:`master configuration
         <master-config-reference>` file.

      .. note::

         If ``scim_enabled`` is ``true``, then ``oidc.auto_provision_users`` must be ``false``.

      For example:

      .. code:: yaml

         oidc:
             enabled: true
             provider: "Okta"
             idp_recipient_url: "https://determined.example.com/oidc/sso"
             idp_sso_url: "https://dev-00000000.okta.com"
             client_id: "xx0xx0"
             client_secret: "xx0xx0"
             auto_provision_users: true
             display_name_attribute_name: "XYZ"

   .. tab::

      SAML

      -  Enable SAML.
      -  Set ``saml.auto_provision_users`` option to ``true`` in your :ref:`master configuration
         <master-config-reference>` file.

      .. note::

         If ``scim_enabled`` is ``true``, then ``saml.auto_provision_users`` must be ``false``.

      For example:

      .. code:: yaml

         saml:
             enabled: true
             provider: "Okta"
             idp_recipient_url: "https://determined.example.com/saml/sso"
             idp_sso_url: "https://myorg.okta.com/app/...sso/saml"
             idp_metadata_path: "https://myorg.okta.com/app/.../sso/saml/metadata"
             auto_provision_users: true

Determined sets the username of the user to the IdP email address. You cannot set the username
independently.

Set the Groups Claim Name Option
================================

Determined receives OIDC and SAML claims via a JSON Web Token (JWT) that you send from your IdP. If
there is a group that does not already exist in Determined, then the system creates the group.

To enable group membership synchronization:

.. tabs::

   .. tab::

      OIDC

      -  Set the ``groups_attribute_name`` option to match the claim name for group memberships from
         your authenticator (i.e., either a custom user attribute such as ``groups_memberships`` or
         ``usergroup_memberships``, or the authenticator's groups claim, passed in as ``groups``,
         etc.).

      For example, in the following claim, let's say the user-group information is passed through
      ``group_memberships`` in your IdP.

      .. code::

         {
           email: dee.ray@example.com
           group_memberships: ['A', 'B'],
           displayName: "Dee Ray"
          }

      Then, Determined creates the following user:

      .. code::

         {
          username: dee.ray@example.com
          groups: ['A', 'B'],
          displayName: "Dee Ray"
         }

   .. tab::

      SAML

      -  Set the ``groups_attribute_name`` option to match the claim name for group memberships from
         your authenticator (i.e., ``groups_memberships``, ``usergroup_memberships``, etc.).

Each time the authenticated user accesses Determined, their information is passed to Determined, and
group memberships are updated. For example, when a user is assigned to a new group via your IdP,
that information is updated in Determined.

Complete the Auto Provision Process
===================================

Once auto provisioning is configured, the user simply signs in with their username.

For example, to sign in to Determined via Okta, the user performs the following steps:

-  Visit the Determined URL, e.g., ``https://determined.example.com``.
-  Under **Sign in with Okta**, enter the SSO-enabled email address.

If the sign in is successful, Determined provisions the user, adds the user to the user table, and
authenticates the user to Determined.

**********************************************
 Manage Remote Users and Groups via the WebUI
**********************************************

To manage users and groups in the WebUI:

-  Sign in to the Determined cluster via the Determined URL, e.g.,
   ``https://determined.example.com``, as a cluster admin.
-  View **Admin Settings** by selecting your profile in the upper left corner and then choosing
   **Admin**.

Actions you can take include adding new users, switching existing users to remote users, performing
bulk actions, and more. For example, using the filters, you can select to view only active users.
You can also manage user groups.

To find out how to manage remote users via the WebUI, including adding a new remote user, visit
:ref:`managing RBAC via the WebUI <manage-users-groups-webui>`.
