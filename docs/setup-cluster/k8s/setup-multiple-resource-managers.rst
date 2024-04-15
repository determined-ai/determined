.. _multiple-resource-managers:

#################################################
 DRAFT ONLY Configure Multiple Resource Managers
#################################################

.. meta::
   :description: Discover how to configure and manage multiple resource managers.

Multi-RM for Kubernetes allows users to set up a Determined master service in one Kubernetes cluster
and schedule workloads in the same or other Kubernetes clusters. - Resource pools are in a
many-to-one relationship with resource managers. - No one resource pool will span multiple resource
managers.

.. attention::

   Resource pool and resource manager names must be unique, among all pools and managers. Otherwise,
   the cluster will crash. Additional resource managers are required to have at least one resource
   pool defined.

**********
 Overview
**********

MultiRM for Kubernetes adds the ability to set up the Determined master service on one Kubernetes
cluster and manage workloads across different Kubernetes clusters. Additional non-default resource
managers and resource pools are configured under the master configuration options
``additional_resource_managers`` and ``resource_pools``. On the WebUI, view the resource manager
name for resource pools. Any requests to resource pools not defined in the master configuration to
the default resource manager, not any additional resource manager, if defined.

*********************************************
 How to Configure Multiple Resource Managers
*********************************************

How To Configure (For Admins)

-  Modify Master Configuration File:

-  To set up multiple resource managers for Kubernetes, start by editing the master configuration
   file. The default resource manager will be in place without requiring a specific name.

-  Configuration Structure:

   -  Locate the resource_manager section in the yaml file; this represents the default resource
      manager.
   -  Add ``additional_resource_managers`` under the resource_manager to configure extra resource
      managers.
   -  Under ``additional_resource_managers``, define resource_pools for each additional resource
      manager.

-  Naming and Rules:

   -  Each resource manager under ``additional_resource_managers`` must have a unique name; failure
      to do so will cause the cluster to crash.
   -  Ensure each additional resource manager has at least one resource pool defined.
   -  Resource pool names must be unique across the cluster to prevent crashes.

**********
 Examples
**********

-  Setting the master configuration (devcluster): - Create as many resource managers (clusters) as
   you’d like. - For each cluster, note each kubeconfig location for the cluster (this is where the
   credentials are found). - Copy or modify the default devcluster template at
   tools/devcluster.yaml. - In the copied devcluster.yaml file, under the master configuration, set
   one of your resource managers as the default, and the rest under additional_resource_managers:

   .. code:: yaml

      resource_manager:
      type: kubernetes
      name: default-rm # optional, should match the name of your default RM/cluster
      ... add whatever other specs you might need ...
      additional_resource_managers:
      - resource_manager:
         type: kubernetes
         name: additional-rm # should match the name of your other RM(s)
         kubeconfig_path: <whatever-path-your-rm-config-is-like ~/.kubeconfig>
         ... add whatever other specs you might need ...
         resource_pools:
            - pool_name: <your-rm-pool-name>

   -  Run the new devcluster: ``devcluster -c <path-to-modified-devcluster>``.

-  Setting the master configuration (Helm): To deploy MultiRM on Kubernetes through a Helm chart,
   the cluster administrator will have to load the credentials for each additional cluster through a
   Kubernetes secret. Follow these steps for each additional resource manager you want to add, and
   then apply the Helm chart once at the end. Let rm-name be the same as the “cluster name” for a
   given cluster.

   -  Set up your additional clusters. These can be from the same or different clouds (i.e., GKE,
      AKS, EKS).

   -  Gather the credentials for each cluster: - For example: .. code:: bash

         # for AKS az aks get-credentials --resource-group <resource-gp-name> --name <rm-name> # for
         GKE gcloud container clusters get-credentials <rm-name>

   -  Using the cluster as the current context, save its kubeconfig to some tmp file.

   -  Repeat the above steps as many times as needed for the additional clusters you want to add.

   Then, switch to the cluster/context that you want to use as the default cluster. Then, repeat the
   following steps to create secrets for each additional cluster you want to add.

   -  Create a Kubernetes secret, from the tmp files for each additional cluster.

   -  Specify each additional resource manager, and its kubeconfig secret/location in
      ``helm/charts/determined/values.yaml``: - For example: .. code:: yaml

         additional_resource_managers: - resource_manager:

            type: kubernetes name: <rm-name> namespace: default # or whatever other namespace you
            want to use kubeconfig_secret_name: <The secret name, from ``kubectl describe secret
            <rm-name>``> kubeconfig_secret_value: <The data value, from ``kubectl describe secret
            <rm-name>``> ... and any other specs you may want to configure ... resource_pools:

               -  pool_name: <rm-pool>

   -  Once all of your resource managers are added to helm values file, install the Helm chart.

-  Setting the master IP/Port for different resource managers:

   For resource managers where the master IP/Port is not reachable by the additional resource
   managers, you will need to update your Helm chart values/configuration to match the external IP
   of the default determined deployment. Once the cluster administrator has the master IP of the
   default Determined deployment, all that's necessary is to upgrade the Helm deployment with that
   value as the master IP for the additional clusters.

*******
 WebUI
*******

How to Interact with It in the WebUI (For WebUI Users)

-  Viewing Resource Managers:

   -  In the WebUI, navigate to the cluster view where each resource pool card will now display a
      “Resource Manager Name” field.
   -  This field helps identify whether a resource pool is managed locally or by another manager,
      tagged as “Remote” if defined in the Master Configuration file.

-  Understanding Visibility and Access:

   -  The “Resource Manager Name” field is visible to administrators or users with permissions to
      define multiple resource managers.
   -  Users can view all resource pools along with their respective manager names, which helps in
      distinguishing between local and remote resource pools.

-  Usage Example:

   -  After configuring an additional resource pool named “test”, you can log in to the cluster and
      see both the default and test resource pools.
   -  The Resource Manager Name for the default pool will be “default”, while for the test pool, it
      will appear as “additional-rm” or the name you specified.
