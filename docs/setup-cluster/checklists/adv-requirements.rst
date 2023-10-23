.. _advanced-setup-requirements:

####################################
 Advanced Installation Requirements
####################################

.. meta::
   :description: Before setting up Determined, ensure your system meets these requirements.

Before applying the :ref:`advanced setup checklist <advanced-setup-checklist>`, ensure your system
meets these requirements. This guide is aimed at administrators who are setting up Determined for
their organization for the first time.

*******************
 TLS Configuration
*******************

For a successful TLS setup during installation, follow these guidelines:

-  **Master Certificate Chain**: The master should have a full certificate chain, including the root
   certificate.

-  **Non-Well-Known CA Signed Certificate**: If the master certificate is *not* signed by a
   well-known Certificate Authority (CA), you will need additional configurations:

   -  **Agent Configuration**: Agents should have the master certificate name and the certificate
      file specified in the ``agent.yaml``.

   -  **Client Configuration**: Clients must be set with the environment variables
      ``DET_MASTER_CERT_NAME`` and ``DET_MASTER_CERT_FILE``. Alternatively, they can adopt a
      trust-on-first-use approach.

.. _firewall-rules:

.. _port-reference:

**********************
 Network Connectivity
**********************

Ensure your firewall rules allow traffic to and from the master and agents according to the
following criteria:

-  **Master-Compute Connection**: Compute nodes must connect to the master node on the master's
   configured port.

-  **Inter-Compute Connection**: Compute nodes should connect to each other on any port.

-  **Master-Compute Reverse Connection**: The master node must establish a connection to compute
   nodes on any port.

-  **Docker Image Access**: Compute nodes should either:

   -  Access the Docker image repository.
   -  Already contain the relevant pre-downloaded images.

-  **Checkpoint Storage Access**:

   -  Both compute nodes and the master node should access the desired checkpoint storage.
   -  Optionally, client nodes having checkpoint storage access can lead to better performance.

-  **Database Access**: The master node should connect to PostgreSQL.

-  **User Task Resources**: Compute nodes must reach any network resources necessary for user tasks,
   such as fetching packages from PyPI.

-  **Client-Master Connection**: Client machines should connect to the master node using the
   master's configured port.

.. _internet-access:

*****************
 Internet Access
*****************

Without proper internet access setup, Determined agents may fail to reach Docker Hub or other online
resources critical for your tasks. Set up internet access according to the following criteria:

-  **Agent Access to Docker Hub**: Since the Determined Docker images are hosted on Docker Hub,
   Determined agents need access to Docker Hub for such tasks as building new images for user
   workloads.

-  **Agent Access to Internet Resources**: If packages, data, or other resources needed by user
   workloads are hosted on the public Internet, Determined agents need to be able to access them.
   Note that agents can be :ref:`configured to use proxies <agent-network-proxy>` when accessing
   network resources.

-  **Master and Agent Use Same Physical Network**: For best performance, it is recommended that the
   Determined master and agents use the same physical network or VPC. When using VPCs on a public
   cloud provider, additional steps might need to be taken to ensure that instances in the VPC can
   access the Internet:

   -  On GCP, the instances need to have an external IP address, or a `GCP Cloud NAT
      <https://cloud.google.com/nat/docs/overview>`_ should be configured for the VPC.

   -  On AWS, the instances need to have a public IP address, and a `VPC Internet Gateway
      <https://docs.aws.amazon.com/vpc/latest/userguide/VPC_Internet_Gateway.html>`_ should be
      configured for the VPC.
