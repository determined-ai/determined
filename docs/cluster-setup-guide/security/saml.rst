.. _saml:

##################
 SAML Integration
##################

.. attention::

   SAML integration applies only to Determined Enterprise Edition.

Determined EE provides a SAML integration to allow users to use single sign-on (SSO) with their
organization's identity provider (IdP) and to provide system administrators better control over
access to resources. Currently, the only officially supported identity provider is Okta.

********************
 Configure Your IdP
********************

When configuring your IdP to allow users to SSO to Determined, you will need to specify the location
of Determined's SSO URL and the audience URL. The audience URL should be set to the Determined
master's base URL and the SSO endpoint is at that base URL with a path of ``/saml/sso``.

Determined also requires an additional attribute named ``userName`` with name format ``unspecified``
set to the username of the user attempting SSO (e.g., for Okta, this is the attribute value
``user.login``).

**********************
 Configure Determined
**********************

Determined requires your IdP's SSO URL, descriptor SSO URL, public certificate, and name as well as
the hostname your IdP intends to use to communicate with Determined. These are all supplied in the
``master.yaml``.

*************************
 Example Setup with Okta
*************************

In this example, we assume the Determined master will run at ``https://determined.example.com``.

First, in Okta, you'll need to create a new SAML application and specify the following options:

.. list-table::
   :widths: 25 50
   :header-rows: 1

   -  -  Field
      -  Example Value
   -  -  Single sign on URL
      -  ``https://determined.example.com/saml/sso``
   -  -  Audience URI (SP Entity ID)
      -  ``https://determined.example.com``
   -  -  Name ID format
      -  Unspecified
   -  -  Application username
      -  Okta username

Determined also requires an attribute statement named ``userName`` with the name format unspecified
and a value ``user.login``.

Okta will show that more steps are required to complete the configuration and link to a page with
their IdP SSO URL, IdP Issuer URL (synonymous to IdP Descriptor URL), and public certificate. Use
these to configure the Determined master in ``master.yaml``. The cert will need to be saved and
mounted into the master's container.

.. code:: yaml

   saml:
     enabled: true
     provider: "Okta"
     idp_recipient_url: "https://determined.example.com/saml/sso"
     idp_sso_url: "https://myorg.okta.com/app/.../sso/saml"
     idp_sso_descriptor_url: "http://www.okta.com/..."
     idp_cert_path: "okta.cert"

Once the master is started with this configuration, users should be able to log in to Determined
from Okta by clicking the Determined tile after they have been provisioned.
