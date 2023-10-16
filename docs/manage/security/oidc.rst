.. _oidc:

############################
 OpenID Connect Integration
############################

.. attention::

   OpenID Connect Integration applies only to Determined Enterprise Edition.

Determined EE provides an OpenID Connect (OIDC) integration allowing users to use single sign-on
(SSO) with their organization's identity provider (IdP). OIDC is an extension of OAuth 2.0 which
allows applications to request information about authenticated users.

Note that users can only log in via OpenID Connect if they have already been provisioned into
Determined. This can be done manually, or via SCIM.

********************
 Configure Your IdP
********************

When configuring your IdP to allow users to SSO to Determined, you will need to specify the location
of Determined's callback URL. This is the URL to which users will be redirected after
authentication.

The callback URL should be set to the Determined master's base URL with a path of
``/oidc/callback``.

**********************
 Configure Determined
**********************

Determined requires your IdP's SSO URL and name, the client id and client secret provided to you by
your IdP, and the public hostname of the master. These are all configured in ``master.yaml``.

Many IdPs require their callback to be sent over HTTPS. If this is the case for your IdP, you should
:ref:`configure the master to use TLS <tls>`.

*************************
 Example Setup with Okta
*************************

In this example, we assume the Determined master will run at ``https://determined.example.com``.

First, in Okta, you'll need to create a new App Integration. You should select OIDC as the sign-in
method and Web Application as the application type.

Then configure the following options:

.. list-table::
   :widths: 25 50
   :header-rows: 1

   -  -  Field
      -  Example Value
   -  -  App Integration Name
      -  My Determined Cluster
   -  -  Allowed Callback URLs
      -  ``https://determined.example.com/oidc/callback``
   -  -  Sign-out redirect URIs
      -  ``https://determined.example.com/det/logout``

Take note of the Domain, Client ID, and Client Secret. You will need to add these to the Determined
Determined master configuration in ``master.yaml``. The Domain corresponds to the ``idp_sso_url``
field.

.. code:: yaml

   oidc:
     enabled: true
     provider: "Okta"
     idp_recipient_url: "https://determined.example.com"
     idp_sso_url: "https://dev-00000000.okta.com"
     client_id: "xx0xXXXxxxXxXXXxXXX0XxX0XXxXXxXX"
     client_secret: "Xxx0xXXXxxXXXxXXxxXX0xxxXXxxxXXxXXXXxXXXxXxXXxxXXXX0XXxXxX-XX0-X"

Once the master is started with this configuration, users will be able to log in to Determined by
clicking the 'Sign in with Okta' button on the sign-in page.
