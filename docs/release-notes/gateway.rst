:orphan:

**New Features**

-  Kubernetes: Introduced the Internal Task Gateway feature which allows Determined tasks running on
   remote K8s clusters to be exposed to the Determined master and proxies. This feature supports
   multi-resource manager setups by configuring a Gateway controller in the external K8s cluster.

   -  **Important:** If set up this feature exposes Determined tasks to outside the cluster world.
      Ensure that you have the necessary security measures in place to limit access to the exposed
      tasks and secure communication between the external cluster and the main one. Recommended
      measures include setting up a firewall, using a VPN, IP white-listing, K8s Network Policies,
      or other security measures.
