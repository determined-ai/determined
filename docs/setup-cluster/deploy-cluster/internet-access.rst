:orphan:

.. _internet-access:

###########################
 Setting Up Internet Access
###########################

-  The Determined Docker images are hosted on Docker Hub. Determined agents need access to Docker
   Hub for such tasks as building new images for user workloads.

-  If packages, data, or other resources needed by user workloads are hosted on the public Internet,
   Determined agents need to be able to access them. Note that agents can be :ref:`configured to use
   proxies <agent-network-proxy>` when accessing network resources.

-  For best performance, it is recommended that the Determined master and agents use the same
   physical network or VPC. When using VPCs on a public cloud provider, additional steps might need
   to be taken to ensure that instances in the VPC can access the Internet:

   -  On GCP, the instances need to have an external IP address, or a `GCP Cloud NAT
      <https://cloud.google.com/nat/docs/overview>`_ should be configured for the VPC.

   -  On AWS, the instances need to have a public IP address, and a `VPC Internet Gateway
      <https://docs.aws.amazon.com/vpc/latest/userguide/VPC_Internet_Gateway.html>`_ should be
      configured for the VPC.
