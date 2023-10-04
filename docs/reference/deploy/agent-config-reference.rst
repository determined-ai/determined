.. _agent-config-reference:

###############################
 Agent Configuration Reference
###############################

*****************
 ``config_file``
*****************

Path to the agent configuration file. Normally this should only be set via an environment variable
or command-line option. Defaults to ``/etc/determined/agent.yaml``.

*****************
 ``master_host``
*****************

Required. The hostname or IP address of the Determined master.

*****************
 ``master_port``
*****************

The port of the Determined master. Defaults to ``443`` if TLS is enabled and ``80`` otherwise.

**************
 ``agent_id``
**************

The ID of this agent; defaults to the hostname of the current machine. Agent IDs must be unique
within a cluster.

***************************
 ``container_master_host``
***************************

Master hostname that containers started by this agent will connect to. Defaults to the value of
``master_host``.

***************************
 ``container_master_port``
***************************

Master port that containers started by this agent will connect to. Defaults to the value of
``master_port``.

*******************
 ``resource_pool``
*******************

Which resource pool the agent should join. Defaults to the value of ``default``, which will work if
and only if there is a resource pool named ``default``. For more information please see
:ref:`resource-pools`.

******************
 ``visible_gpus``
******************

The GPUs that should be exposed as slots by the agent. A comma-separated list of GPUs, each
specified by a 0-based index, UUID, PCI bus ID, or board serial number. The 0-based index of NVIDIA
GPUs or AMD GPUs can be obtained via the ``nvidia-smi`` or ``rocm-smi`` commands.

***************
 ``slot_type``
***************

The slot type that should be exposed. Dynamic agents having GPUs will be configured to ``cuda``,
agents without GPUs with ``cpu_slots_allowed: true`` provisioner option will be configured to
``cpu``, and ``none`` otherwise. For static agents this field defaults to ``auto``.

-  ``auto``: Automatically detects the slot type. The agent will detect if there are NVIDIA GPUs or
   AMD GPUs. If there are GPUs, it maps each GPU to one slot. Otherwise, it maps all the CPUs to a
   slot.

``none``: The agent will not create any slots for detected devices.

``cuda``: The agent will map each detected NVIDIA GPU to a slot. Prior to Determined 0.17.6, this
option was called ``gpu``.

``cpu``: Map all the CPUs to a slot, even when GPUs are present.

``rocm``: The agent will map each detected ROCm AMD GPU to a slot.

****************
 ``http_proxy``
****************

The HTTP proxy address for the agent's containers.

*****************
 ``https_proxy``
*****************

The HTTPS proxy address for the agent's containers.

***************
 ``ftp_proxy``
***************

The FTP proxy address for the agent's containers.

**************
 ``no_proxy``
**************

The addresses that the agent's containers should not proxy.

**************
 ``security``
**************

Security-related configuration settings.

``tls``
=======

Configuration settings for :ref:`TLS <tls>`.

-  ``enabled``: Whether to use TLS to connect to the master. Defaults to ``false``.
-  ``skip_verify``: Skip verifying the master certificate when using TLS. Defaults to ``false``.
   Enabling this setting will reduce the security of your Determined cluster.
-  ``master_cert``: CA cert file for the master when using TLS.
-  ``master_cert_name``: A hostname for which the master's TLS certificate is valid, if the value of
   the ``master_host`` option is an IP address or is not contained in the certificate.
-  ``client_cert``/``client_key``: Paths to files containing the client TLS certificate and key to
   use when connecting to the master.

************
 ``fluent``
************

fluentd settings.

``image``
=========

Docker image to use for the managed Fluent Bit daemon. Defaults to ``fluent/fluent-bit:1.9.3``.

``port``
========

TCP port for the Fluent Bit daemon to listen on. Defaults to port 24224. Should be unique when
running multiple agents on the same node.

``container_name``
==================

Name for the Fluent Bit container. Defaults to ``determined-fluent``. Should be unique when running
multiple agents on the same node.

******************************
 ``agent_reconnect_attempts``
******************************

Maximum number of times the agent will attempt to reconnect to master on connection failure.
Defaults to 5.

*****************************
 ``agent_reconnect_backoff``
*****************************

Time interval between reconnection attempts, in seconds. Defaults to 5 seconds.

********************************************
 ``container_auto_remove_disabled`` (debug)
********************************************

Whether to disable setting ``AutoRemove`` flag on task containers. Defaults to false.

***********
 ``hooks``
***********

Configuration for commands to run when certain events occur. The value of each option in this
section is an array of strings specifying the command and its arguments.

``on_connection_lost``
======================

A command to run when the agent fails to either connect to the master on startup or reconnect after
a loss of connection. When reconnecting, the agent will make several attempts as specified by the
``agent_reconnect_attempts`` and ``agent_reconnect_backoff`` configuration options.

In order to shut down the machine on which the agent is running, set this to ``["sudo", "shutdown",
"now"]``, or just ``["shutdown", "now"]`` if the agent is running as root. Additional system
configuration may be required in order to allow the agent to execute the command from inside a
Docker container or without the need to enter a password.

***********
 ``label``
***********

Deprecated. This field has been deprecated and will be ignored. Use ``resource_pool`` instead.

***********
 ``debug``
***********

If ``true``, enables a more verbose form of logging that may be helpful in diagnosing issues.
Defaults to ``false``.

****************
 ``image_root``
****************

If set then specifies the path to a shared directory of previously downloaded Determined environment
images. If not defined, then Determined environments will be downloaded automatically. For more
information on setting up an image cache see :ref:`singularity-image-cache`. Defaults to undefined.
