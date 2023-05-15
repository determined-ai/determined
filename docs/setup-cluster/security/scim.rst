.. _scim:

##################
 SCIM Integration
##################

.. attention::

   SCIM integration applies only to Determined Enterprise Edition.

Determined EE provides a System for Cross-domain Identity Management (SCIM) integration to allow
administrators to easily and securely provision users and groups through their standard identity
provider (IdP). Currently, the only officially supported provider is Okta; however, Determined
implements a minimal working subset of the protocol as specified by :RFC:`7644` and is expected to
work with most IdPs that adhere to this RFC.

**********************
 Configure Determined
**********************

Determined only requires you to enable SCIM and set your authentication mode and any necessary
credentials.

********************
 Configure Your IdP
********************

When configuring your IdP to automatically push users and groups to Determined, you will need to
enable SCIM for Determined in your IdP and provide it with some information about Determined's SCIM
API. This includes the SCIM connector base URL, the unique ID for users that are provisioned, which
provisioning actions should be supported, and the authentication mode to use.

*************************
 Example Setup with Okta
*************************

In this example, we assume the Determined master will run at ``https://determined.example.com`` and
that we are using the basic authentication mode. For more information on configuring Determined to
use the OAuth2 authentication mode, see the :ref:`OAuth <oauth>` topic guide.

First, in Okta, you'll need to create a new Okta application for Determined or alter your existing
one. Under ``Determined -> General -> App Settings -> Provisioning``, select the ``SCIM`` radio
button. A new tab ``Determined -> Provisioning`` should appear---navigate to its ``Integration`` tab
and specify the following configurations for the integration:

.. list-table::
   :widths: 25 50
   :header-rows: 1

   -  -  Field
      -  Example Value
   -  -  SCIM connector base URL
      -  ``https://determined.example.com/scim/v2``
   -  -  Unique identifier for users
      -  ``userName``
   -  -  Supported provisioning actions
      -  Check all boxes your organization wants to support
   -  -  Authentication Mode
      -  Basic Auth
   -  -  Username
      -  ``determined``
   -  -  Password
      -  ``password``

.. note::

   The username and password shown here are arbitrary and only expected to match the values
   specified in the ``scim.auth`` master configuration.

Then navigate to ``Determined -> Provisioning -> To App`` and enable the provisioning features your
organization wants to use with Okta.

To configure Determined for this integration, update the :ref:`master configuration
<master-config-reference>` to provide the client's credentials and enable the SCIM server.

.. code:: yaml

   scim:
     enabled: true
     auth:
       type: basic
       username: "determined"
       password: "password"
