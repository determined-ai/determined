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

.. _tls-agent-config:

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

This section describes how to set up a TLS certificate from `Let's Encrypt
<https://letsencrypt.org>`__ using `Certbot <https://certbot.eff.org/>`__ and either the HTTP-01 or
the DNS-01 challenge type.

.. note::

   For more information about the challenge types, visit the `Let's Encrypt documentation
   <https://letsencrypt.org/docs/challenge-types/>`_.

Installing snapd and Certbot
============================

This section provides information about installing snapd and Certbot and adding EPEL to RHEL 8 or
CentOS 8.

The following websites provide more information about installing snapd and Certbot:

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

To add the EPEL repository to a CentOS Stream 8/9 system, run the following commands:

.. code:: bash

   sudo dnf install epel-release
   sudo dnf upgrade

Installing snapd
----------------

To install snapd, run the following commands:

.. code:: bash

   sudo yum install snapd
   sudo systemctl enable --now snapd.socket
   sudo ln -s /var/lib/snapd/snap /snap

Installing Certbot
------------------

To install Certbot on RHEL or CentOS, run the following command:

.. code:: bash

   sudo snap install --classic certbot

To install Certbot on Debian/Ubuntu, run the following command:

.. code:: bash

   sudo apt-get install certbot

Certbot Certificate Request
===========================

To complete the Certbot certificate request, execute the following steps as the root user:

-  Register a Let's Encrypt account
-  Perform a certificate request
-  Update the Determined master configuration to use the certificate

The steps are described in detail in the following sections.

Register a Let's Encrypt Account
--------------------------------

To register an account on Let's Encrypt, run the following command:

.. code:: bash

   certbot register

Certbot responds letting you know the account is registered.

To check the account status, run the following command:

.. code:: bash

   certbot show_account

Certbot responds with the account details including the account URL, thumbprint, and email contact.

Perform a Certificate Request
-----------------------------

Certificate Creation
^^^^^^^^^^^^^^^^^^^^

If port 80 of the Determined Master is accessible, you can use a simple `HTTP-01 challenge
<https://letsencrypt.org/docs/challenge-types/#http-01-challenge>`_ type.

Certificate Creation When the Determined Master is Behind a VPN
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

This section provides information about requesting the Let's Encrypt certificate in environments
that do not provide inbound access from Let's Encrypt to port 80 of the Determined master (e.g.,
when the Determined master is behind a VPN).

Request a Certificate Using the DNS-01 Challenge
""""""""""""""""""""""""""""""""""""""""""""""""

Run the following command to request a Let's Encrypt certificate using the DNS-01 challenge type:

.. code:: bash

   certbot certonly --manual --preferred-challenges dns -d <domain>

Certbot responds with a domain token and lets you know that before continuing you should verify that
the TXT record has been deployed:

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

   Do not press **Enter** before setting up the DNS record.

Set Up the DNS Record
"""""""""""""""""""""

In the DNS configuration for the domain the Determined master is using, create a record with the
following values:

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

   You may need to install the ``nslookup`` utility.

   On CentOS:

   .. code:: bash

      yum install bind-utils

   On Debian/Ubuntu:

   .. code:: bash

      apt install dnsutils

Complete the Certificate Request
""""""""""""""""""""""""""""""""

Once you have set up the DNS record, press **Enter**.

Certbot lets you know it has received the certificate and provides the certificate location, key
location, and certificate expiration date.

Update the Determined Master TLS Configuration
----------------------------------------------

This section describes how to update the Determined master configuration to use the TLS certificate
provided by the Let's Encrypt service.

First, stop the Determined master using the appropriate command. For example, if you installed
Determined using Linux packages, run the following command:

.. code:: bash

   systemctl stop determined-master

Then, change the security section of the master configuration file by adding the following lines:

.. code:: yaml

   security:
      tls:
         cert: /etc/letsencrypt/live/<domain>/fullchain.pem
         key: /etc/letsencrypt/live/<domain>/privkey.pem

If appropriate, change the master port:

.. code:: yaml

   port: 443

.. important::

   You'll need to configure the agents to reach this port.

Finally, start the Determined master using the appropriate command. For example, if you installed
Determined using Linux packages, run the following command:

.. code:: bash

   systemctl start determined-master

Certbot Certificate Renewal
===========================

To renew the certificate, repeat the certificate creation steps, and restart the Determined master
using the appropriate command. For example, if you installed Determined using Linux packages, run
the following command:

.. code:: bash

   systemctl restart determined-master

.. note::

   Most Certbot installations come with automatic renewal. Visit `Setting up automated renewals
   <https://eff-certbot.readthedocs.io/en/stable/using.html#automated-renewals>`__ to find out more.
   To learn how to test automatic renewal, visit the Certbot instructions (`CentOS
   <https://certbot.eff.org/instructions?ws=other&os=centosrhel8>`__ or `Debian/Ubuntu
   <https://certbot.eff.org/instructions?ws=apache&os=ubuntufocal>`__).
