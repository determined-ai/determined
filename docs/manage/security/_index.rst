.. _security-overview:

##########
 Security
##########

These security features apply only to Determined Enterprise Edition, except where noted.

+-----------------------+-------------------------------------------------------------------------+
| Security Feature      | Description                                                             |
+=======================+=========================================================================+
| :ref:`oauth`          | Enable, list, and remove OAuth clients.                                 |
+-----------------------+-------------------------------------------------------------------------+
| :ref:`tls`            | Set up the master and agents to use TLS security. This feature is       |
|                       | available for both Determined Open Source and Enterprise editions.      |
+-----------------------+-------------------------------------------------------------------------+
| :ref:`oidc`           | Integrate OpenID Connect, with an Okta example.                         |
+-----------------------+-------------------------------------------------------------------------+
| :ref:`saml`           | Integrate Security Assertion Markup Language (SAML) authentication to   |
|                       | use single sign-on (SSO) with your organization's identity provider     |
|                       | (IdP).                                                                  |
+-----------------------+-------------------------------------------------------------------------+
| :ref:`scim`           | Integrate System for Cross-domain Identity Management (SCIM) for        |
|                       | administrators to easily and securely provision users and groups.       |
+-----------------------+-------------------------------------------------------------------------+
| :ref:`rbac`           | Configure Role-Based Access Control.                                    |
+-----------------------+-------------------------------------------------------------------------+
| :ref:`access-tokens`  | Manage access tokens to enable secure automation of workflows through   |
|                       | API authentication.                                                     |
+-----------------------+-------------------------------------------------------------------------+
| :ref:`auto-posix`     | Configure automatic POSIX user linking based on OIDC/SAML claims.       |
+-----------------------+-------------------------------------------------------------------------+

.. toctree::
   :maxdepth: 1
   :hidden:
   :glob:

   ./*
