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
