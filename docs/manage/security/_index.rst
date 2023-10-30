.. _security-overview:

##########
 Security
##########

These security features apply only to Determined Enterprise Edition, except for TLS.

+-------------------+----------------------------------------------------------------------------+
| Security Feature  | Description                                                                |
+===================+============================================================================+
| :ref:`oauth`      | Enable, list, and remove OAuth clients.                                    |
+-------------------+----------------------------------------------------------------------------+
| :ref:`tls`        | Set up the master and agents to use TLS security.                          |
+-------------------+----------------------------------------------------------------------------+
| :ref:`oidc`       | Integrate OpenID Connect, with and Okta example.                           |
+-------------------+----------------------------------------------------------------------------+
| :ref:`saml`       | Integrate Security Assertion Markup Language (SAML) authentication to use  |
|                   | single sign-on (SSO) with your organizationidentity provider (IdP).        |
+-------------------+----------------------------------------------------------------------------+
| :ref:`scim`       | Integrate System for Cross-domain Identity Management (SCIM) for           |
|                   | administrators to easily and securely provision users and groups.          |
+-------------------+----------------------------------------------------------------------------+
| :ref:`rbac`       | Configure Role-Based Access Control.                                       |
+-------------------+----------------------------------------------------------------------------+

.. toctree::
   :maxdepth: 1
   :hidden:
   :glob:

   ./*
