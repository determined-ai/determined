.. _tls:

##########################
 Transport Layer Security
##########################

**Transport Layer Security** (TLS) is a protocol for secure network communication. TLS prevents the
data being transmitted from being modified or read while it is in transit and allows clients to
verify the identity of the server (in this case, the Determined master). Determined can be
configured to use TLS for all connections made to the master. That means that all CLI and WebUI
connections will be secured by TLS, as well as connections from agents and tasks to the master.
Communication between agents that occur as part of :ref:`distributed training <multi-gpu-training>`
will not use TLS, nor will proxied connections from the master to a :ref:`TensorBoards
<tensorboards>` or :ref:`notebook <notebooks>` instance.

After the master and agent are configured to use TLS, no additional configuration is needed for
tasks run in the cluster. In shells and notebooks, the Determined Python libraries automatically
make connections to the master using TLS with the appropriate certificate.

**********************
 Master Configuration
**********************

To :ref:`configure the master <master-config-reference>` to use TLS, set the ``security.tls.cert``
and ``security.tls.key`` options to paths to a TLS certificate file and key file.

When TLS is in use, the master will listen on TCP port 8443 by default, rather than 8080.

.. note::

   If the master's certificate is not signed by a well-known CA, then the configured certificate
   file must contain a full certificate chain that goes all the way to a root certificate.

**********************
 Agents Configuration
**********************

When the Determined master is using TLS, set the ``security.tls.enabled`` :ref:`agent configuration
option <agent-config-reference>` to ``true``. If the master's certificate is signed by a well-known
CA, then no other TLS-specific configuration is necessary. Otherwise, for the best security, place
the master's certificate file somewhere accessible to the agent and set the agent's
``security.tls.master_cert`` option to the path to that file. For a more convenient but less secure
setup, instead set the ``security.tls.skip_verify`` option to ``true``. With the latter
configuration, the agent will be unable to verify the identity of the master, but the data sent over
the connection will still be protected by TLS.

If the master's certificate does not contain the address that the agent is using to connect to the
master (but is otherwise valid), set the ``security.tls.master_cert_name`` option to one of the
addresses in the certificate. For example, the master's certificate may contain a DNS hostname
corresponding to the public IP address of the master, while the agent connects to the master using
its private IP address to prevent traffic from being routed over the public Internet. In that case,
the option should be set to the DNS name contained in the certificate.

.. note::

   Due to a limitation of `Fluent Bit <https://fluentbit.io>`__, which Determined uses internally,
   the certificate must be valid for at least one hostname that is not an IP address and the
   ``security.tls.master_cert_name`` option must be set to that hostname if the agent is configured
   to connect to the master using an IP address. The hostname does not need to be an actual DNS name
   for the master---it is only used for certificate verification.

When :ref:`dynamic agents <elastic-infrastructure>` and TLS are both in use, the dynamic agents that
the master creates will automatically be configured to connect securely to the master over TLS.

*******************
 CLI Configuration
*******************

To use TLS, the CLI must be configured with a master address starting with ``https://`` using either
the ``-m`` flag or ``DET_MASTER`` environment variable.

If the master's certificate is signed by a well-known CA, then the connection should proceed
immediately. If not, the CLI will indicate on the first connection that the master is presenting an
untrusted certificate and display a hash of the certificate. You may wish to confirm the hash with
your system administrator; in any case, if you confirm the connection to the master, the certificate
will be stored on the computer where the CLI is being run and future connections to the master will
be made without confirmation.

*************************************
 Let's Encrypt TLS Certificate Setup
*************************************

This section describes how to set up the Let's Encrypt TLS certificate.

Installing a Let's Encrypt TLS certificate requires an ACME protocol client ``certbot`` and one of
the following domain token verification methods:

-  HTTP-01, or
-  DNS-01

.. note::

   For more information about the domain token verification methods, visit `Let's Encrypt
   documentation <https://letsencrypt.org/docs/challenge-types/>`_.

Prerequisites
=============

You must have ``snapd`` installed to use ``certbot``.

Installing Snapd and Certbot
============================

This section provides information about installing ``snapd`` and ``certbot`` and adding EPEL to RHEL
8 or CentOS 8.

The following websites provide more information about installing ``snapd`` and ``certbot``:

-  `Installing snap on Red Hat Enterprise Linux (RHEL)
   <https://snapcraft.io/docs/installing-snap-on-red-hat>`_
-  `Installing snap on CentOS <https://snapcraft.io/docs/installing-snap-on-centos>`_
-  `certbot instructions <https://certbot.eff.org/instructions?ws=other&os=centosrhel8>`_

Adding EPEL to RHEL 8
---------------------

To add the EPEL repository to a RHEL 8 system, run the following commands:

.. code:: bash

   sudo dnf install https://dl.fedoraproject.org/pub/epel/epel-release-latest-8.noarch.rpm
   sudo dnf upgrade

Adding EPEL to CentOS 8
-----------------------

To add the EPEL repository to a CentOS 8/9 Stream system, run the following commands:

.. code:: bash

   sudo dnf install epel-release
   sudo dnf upgrade

Installing Snapd Software
-------------------------

To install ``snapd``, run the following commands:

.. code:: bash

   sudo yum install snapd
   sudo systemctl enable --now snapd.socket
   sudo ln -s /var/lib/snapd/snap /snap

.. note::

   On Debian/Ubuntu, ``snapd`` is usually already installed.

Installing Certbot Software
---------------------------

To install ``certbot``, run the following command:

.. code:: bash

   sudo snap install --classic certbot

To install ``certbot`` on Debian/Ubuntu, run the following command:

.. code:: bash

   sudo apt-get install certbot

Certbot Certificate Request
===========================

To complete the ``certbot`` certificate request, execute the following steps as root user:

-  Account registration
-  Manual certificate request
-  MLDE Master configuration to point to the certificate

The steps are described in detail in the following sections.

Account Registration
--------------------

To register the account on Let's Encrypt, run the following command:

.. code:: bash

   # certbot register

Certbot responds letting you know the account is registered.

To check the account status, run the following command:

.. code:: bash

   # certbot show_account

Certbot responds with the account details including the account URL, thumbprint, and email contact.

Certificate Creation When the Determined Master is Behind a VPN
---------------------------------------------------------------

This section provides information about requesting the Let's Encrypt certificate in environments
that do not provide inbound access from Let’s Encrypt to port Determined Master server port 80
(e.g., when Determined Master is behind a VPN).

Request the Let's Encrypt Certificate using the DNS-01 Challenge
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

Run the following command to request the Let's Encrypt certificate using the DNS-01 challenge domain
verification:

.. code:: bash

   # certbot certonly --manual --preferred-challenges dns -d <domain>

Certbot responds and lets you know that before continuing you should verify the TXT record has been
deployed:

.. code::

   Saving debug log to /var/log/letsencrypt/letsencrypt.log
   Requesting a certificate for <domain>

   Please deploy a DNS TXT record under the name:

   _acme-challenge.<domain>.

   with the following value:

   <XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX domain token>

   Before continuing, verify the TXT record has been deployed. Depending on the DNS
   provider, this may take some time, from a few seconds to multiple minutes. You can
   check if it has finished deploying with the aid of online tools, such as the Google
   Admin Toolbox: https://toolbox.googleapps.com/apps/dig/#TXT/_acme-challenge.<domain>.
   Look for one or more bolded line(s) below the line ';ANSWER'. It should show the
   value(s) you've just added.

   Press Enter to Continue

.. caution::

   DO NOT PRESS ENTER BEFORE SETTING UP THE DNS RECORD

Set Up the DNS Record Before Pressing ``ENTER``
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

DNS TXT Record example

+---------------------------------+-------------+-----+-------------------------------------------------+
| FQDN                            | RECORD TYPE | TTL | Value                                           |
+=================================+=============+=====+=================================================+
| _acme-challenge.<domain>.       | TXT         | 900 | <XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX  |
|                                 |             |     | domain token>                                   |
+---------------------------------+-------------+-----+-------------------------------------------------+

Ensure the ``_acme-challenge.<domain>.`` DNS record has been propagated using one of the following:

-  ``https://toolbox.googleapps.com/apps/dig/#TXT/_acme-challenge.<domain>.``, or
-  ``nslookup -type=TXT _acme-challenge.<domain>.``

.. note::

   You may need to install bind-utils to run ``nslookup`` using the following command:

   .. code:: bash

      yum install bind-utils

Press ``ENTER``
^^^^^^^^^^^^^^^

Once you have set up the DNS record, press **Enter**.

Certbot responds, letting you know it has received the certificate. Certbot provides the certificate
location, key and the certificate expiration date.

.. Important::

   To renew the certificate repeat the certificate creation steps.

Determined Master TLS Configuration
===================================

This section describes how to use the TLS certs provided by the Let’s Encrypt service.

First, stop the Determined Master with the following command:

.. code:: bash

   systemctl stop determined-master

Then, change the security branch of the ``master.yaml`` by adding the following configuration:

.. code:: bash

   security:

   tls:

      cert: /etc/letsencrypt/live/<domain>/fullchain.pem

      key: /etc/letsencrypt/live/<domain>/privkey.pem

Eventually, change the master port:

.. code:: bash

   # master port
   port: 443

.. Important::

   You'll need to configure the agents to reach this port.

Finally, start the Determined Master with the following command:

.. code:: bash

   systemctl start determined-master

After a certificate renewal, you must restart the Determined Master using the following command:

.. code:: bash

   systemctl restart determined-master
