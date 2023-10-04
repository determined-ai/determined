:orphan:

.. _firewall-rules:

################
 Firewall Rules
################

DRAFT The firewall rules must satisfy the following network access requirements for the master and
agents.

********
 Master
********

NEEDS UPDATES

-  Inbound TCP to the master's network port from the Determined agent instances, as well as all
   machines where developers want to use the Determined CLI or WebUI. The default port is ``8443``
   if TLS is enabled and ``8080`` if not.

-  Outbound TCP to all ports on the Determined agents.

********
 Agents
********

NEEDS UPDATES

-  Inbound TCP from all ports on the master to all ports on the agent.

-  Outbound TCP from all ports on the agent to the master's network port.

-  Outbound TCP to the services that host the Docker images, packages, data, and other resources
   that need to be accessed by user workloads. For example, if your data is stored on Amazon S3,
   ensure the firewall rules allow access to this data.

-  Inbound and outbound TCP on all ports to and from each Determined agent. The details are as
   follows:

   -  Inbound and outbound TCP ports 1734 and 1750 are used for synchronization between trial
      containers.
   -  Inbound and outbound TCP port 12350 is used for internal SSH-based communication between trial
      containers.
   -  When using ``DeepSpeedTrial``, port 29500 is used by for rendezvous between trial containers.
   -  When using ``PyTorchTrial`` with the "torch" distributed training backend, port 29400 is used
      for rendezvous between trial containers
   -  For all other distributed training modes, inbound and outbound TCP port 12355 is used for GLOO
      rendezvous between trial containers.
   -  Trials use OS-selected ephemeral ports for communication via GLOO and NCCL.
   -  Each TensorBoard uses a port in the range 2600–2899
   -  Each notebook uses a port in the range 2900–3199
   -  Each shell uses a port in the range 3200–3599

See also: :ref:`internet-access`.
