.. _multiple-resource-managers:

######################################
 Configure Multiple Resource Managers
######################################

.. meta::
   :description: Discover how to configure and manage multiple resource managers.

.. include:: ../../_shared/attn-enterprise-edition.txt

**********
 Overview
**********

Multiple Resource Managers (Multi-RM) for Kubernetes allows you to set up a Determined master
service in one Kubernetes cluster and schedule workloads in the same or other Kubernetes clusters.

**Resource Pool Relationships**

-  Resource pools have a many-to-one relationship with resource managers.
-  No single resource pool will span multiple resource managers.

Any requests to resource pools not defined in the master configuration are routed to the default
resource manager. Such requests are not routed to additional resource managers, if defined.

To enable use of Determined tasks that rely on Determined proxies in the external-to-master
clusters, set up a gateway as described in the docs :doc:Internal Task Gateway here :doc:`Internal
Task Gateway here <internal-task-gateway>`.

*********************************************
 How to Configure Multiple Resource Managers
*********************************************

Multiple resource managers are defined in the :ref:`master configuration <master-config-reference>`
file. To define multiple resource managers, first define one default resource manager and then
define one or more additional resource managers. Default resource managers are defined using the
``resource_manager`` option, whereas any additional resource managers are defined using the
``addtional_resource_manager`` option.

.. attention::

   **Naming and Rules**

   -  Each resource manager under ``additional_resource_managers`` must have a unique name; failure
      to do so will cause the cluster to crash.
   -  Ensure each additional resource manager has at least one resource pool defined.
   -  Resource pool names must be unique across the cluster to prevent crashes.

.. note::

   If desired, you can apply a `name` to the default resource manager.

#. Locate the ``resource_manager`` section in the :ref:`master configuration
   <master-config-reference>` yaml file. This represents the default resource manager.
#. Add ``additional_resource_managers`` under the ``resource_manager`` to configure extra resource
   managers.
#. Under ``additional_resource_managers``, define ``resource_pools`` for each additional resource
   manager.

Example: Master Configuration (devcluster)
==========================================

Follow this example to create as many resource managers (clusters) as needed.

-  For each cluster, note each ``kubeconfig`` location for the cluster (this is where the
   credentials are found).

-  Copy or modify the default devcluster template at ``tools/devcluster.yaml``.

-  In the copied ``devcluster.yaml`` file, under the ``master configuration``, set one of your
   resource managers as the default, and the rest under ``additional_resource_managers``:

   .. code:: yaml

        resource_manager:
       type: kubernetes
      name: default-rm # optional, should match the name of your default RM/cluster
       ... add any other specs you might need ...
      additional_resource_managers:
      - resource_manager:
         type: kubernetes
         name: additional-rm # should match the name of your other RM(s)
         kubeconfig_path: <whatever-path-your-rm-config-is-like ~/.kubeconfig>
         ... add whatever other specs you might need ...
         resource_pools:
            - pool_name: <your-rm-pool-name>

-  Run the new devcluster: ``devcluster -c <path-to-modified-devcluster>``.

For more information, visit the Determined Kubernetes Developer Guide located in the ``k8s/``
subdirectory of the `Determined GitHub repo
<https://github.com/determined-ai/determined/tree/main/tools/k8s>`__.

Example: Master Configuration (Helm)
====================================

To deploy Multi-RM on Kubernetes through a Helm chart, the cluster administrator must load the
credentials for each additional cluster through a Kubernetes secret. Follow these steps for each
additional resource manager you want to add, and then apply the Helm chart once. Let ``rm-name`` be
the same as the “cluster name” for a given cluster.

-  Set up your additional clusters. These can be from the same or different clouds (e.g., GKE, AKS,
   EKS).

-  Gather the credentials for each cluster.

   For example:

      .. code:: bash

         # for AKS
         az aks get-credentials --resource-group <resource-gp-name> --name <rm-name>

         # for GKE
         gcloud container clusters get-credentials <rm-name>

-  Using the cluster as the current context, save its ``kubeconfig`` to a ``tmp`` file.

-  Repeat the above steps as many times as needed for the additional clusters you want to add.

-  Next, switch to the cluster/context that you want to use as the default cluster. Then, repeat the
   following steps to create secrets for each additional cluster you want to add.

-  Create a Kubernetes secret, from the ``tmp`` files for each additional cluster.

-  Specify each additional resource manager, and its kubeconfig secret/location in
   ``helm/charts/determined/values.yaml``.

-  For example:

   .. code:: yaml

      additional_resource_managers:
      - resource_manager:
         type: kubernetes name: <rm-name> namespace: default
         # or whatever other namespace you want to use
         kubeconfig_secret_name: <The secret name, from ``kubectl describe secret <rm-name>``>
         kubeconfig_secret_value: <The data value, from ``kubectl describe secret <rm-name>``>
         ... and any other specs you may want to configure ...
         resource_pools:
            -  pool_name: <rm-pool>

-  Once all of your resource managers are added to helm values file, install the Helm chart.

.. attention::

   Setting the master IP/Port for different resource managers:

   For resource managers where the master IP/Port is not reachable by the additional resource
   managers, you will need to update your Helm chart values/configuration to match the external IP
   of the default determined deployment. Once the cluster administrator has the master IP of the
   default Determined deployment, all that's necessary is to upgrade the Helm deployment with that
   value as the master IP for the additional clusters.

   If an additional resource manager needs to connect to the Determined master through a gateway
   requiring TLS, ``resource_manager.determined_master_scheme`` should be set to ``https``. If
   ``resource_manager.determined_master_scheme`` is not set ``determined_master_host`` will assume
   ``https`` if the master is terminating TLS and ``http`` otherwise.

*******
 WebUI
*******

In the WebUI, the resource manager's associated cluster name is visible for each resource pool.

To view resource managers:

-  In the WebUI, navigate to the cluster view.
-  Each resource pool card displays **Cluster Name**.

This field helps identify whether a resource pool is managed locally or by another manager, tagged
as “Remote” if defined in the :ref:`master-config-reference` file.

**Usage Scenario**

Let's say your environment contains a resource manager and a resource pool that have both adopted
the name "default" due to their unnamed status, and you configure an additional resource pool named
"additional-rm” with a resource pool named "test". When you sign in to the WebUI, you'll see both
the default and test resource pools. The resource manager's cluster name for the default pool will
display “default”, while the test pool displays "additional-rm" (or the name you specified).

**Visibility and Access**

**Cluster Name** is only visible to administrators or users with permissions to define multiple
resource managers. Users can view all resource pools along with the cluster name of each resource
pool's manager to help distinguish between local and remote resource pools.
