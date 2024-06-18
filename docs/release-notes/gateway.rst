:orphan:

**New Features**

-  Kubernetes: The :ref:`Internal Task Gateway <internal-task-gateway>` feature enables Determined
   tasks running on remote Kubernetes clusters to be exposed to the Determined master and proxies.
   This feature facilitates multi-resource manager setups by configuring a Gateway controller in the
   external Kubernetes cluster.

.. important::

   Enabling this feature exposes Determined tasks to the outside world. It is crucial to implement
   appropriate security measures to restrict access to exposed tasks and secure communication
   between the external cluster and the main cluster. Recommended measures include:

      -  Setting up a firewall
      -  Using a VPN
      -  Implementing IP whitelisting
      -  Configuring Kubernetes Network Policies
      -  Employing other security measures as needed
