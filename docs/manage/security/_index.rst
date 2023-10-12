.. _security-overview:

##########
 Security
##########

These security features apply only to Determined Enterprise Edition, except for TLS.

+-------------------+----------------------------------------------------------------------------+
| Security Feature  | Documentation                                                              |
+===================+============================================================================+
| :doc:`oauth`      | Enable, list, and remove OAuth clients.                                    |
+-------------------+----------------------------------------------------------------------------+
| :doc:`tls`        | Set up the master and agents to use TLS security.                          |
+-------------------+----------------------------------------------------------------------------+
| :doc:`oidc`       | Integrate OpenID Connect, with and Okta example.                           |
+-------------------+----------------------------------------------------------------------------+
| :doc:`saml`       | Integrate Security Assertion Markup Language (SAML) authentication to use  |
|                   | single sign-on (SSO) with your organizationidentity provider (IdP).        |
+-------------------+----------------------------------------------------------------------------+
| :doc:`scim`       | Integrate System for Cross-domain Identity Management (SCIM) for           |
|                   | administrators to easily and securely provision users and groups.          |
+-------------------+----------------------------------------------------------------------------+
| :doc:`rbac`       | Configure Role-Based Access Control.                                       |
+-------------------+----------------------------------------------------------------------------+

.. toctree::
   :maxdepth: 2
   :glob:

   ./*
