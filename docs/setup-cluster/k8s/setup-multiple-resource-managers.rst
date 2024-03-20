.. _multiple-resource-managers:

#################################################
 DRAFT ONLY Configure Multiple Resource Managers
#################################################

.. meta::
   :description: Discover how to configure and manage multiple resource managers.

Short introduction...

.. attention::

   For anything we want to call out to avoid crashing the cluster, etc.

**********
 Overview
**********

We might want to describe the feature here.

*********************************************
 How to Configure Multiple Resource Managers
*********************************************

How To Configure (For Admins)

* 		Modify Master Configuration File:
* 		To set up multiple resource managers for Kubernetes, start by editing the master configuration file. The default resource manager will be in place without requiring a specific name.
* 		Configuration Structure:

    * Locate the resource_manager section in the yaml file; this represents the default resource manager.
    * Add ``additional_resource_managers`` under the resource_manager to configure extra resource managers.
    * Under ``additional_resource_managers``, define resource_pools for each additional resource manager.

* 		Naming and Rules:

    * Each resource manager under ``additional_resource_managers`` must have a unique name; failure to do so will cause the cluster to crash.
    * Ensure each additional resource manager has at least one resource pool defined.
    * Resource pool names must be unique across the cluster to prevent crashes.



*********
Example
*********

Add examples for the following:
- Setting kubeconfig
- Setting masterip/port for the different resource managers
- Multicloud
- Multi gke cluster

******
WebUI
******

How to Interact with It in the WebUI (For WebUI Users)

* 		Viewing Resource Managers:

    * In the WebUI, navigate to the cluster view where each resource pool card will now display a “Resource Manager Name” field.
    * This field helps identify whether a resource pool is managed locally or by another manager, tagged as “Remote” if defined in the Master Configuration file.

* 		Understanding Visibility and Access:

    * The “Resource Manager Name” field is visible to administrators or users with permissions to define multiple resource managers.
    * Users can view all resource pools along with their respective manager names, which helps in distinguishing between local and remote resource pools.

* 		Usage Example:

    * After configuring an additional resource pool named “test”, you can log in to the cluster and see both the default and test resource pools.
    * The Resource Manager Name for the default pool will be “default”, while for the test pool, it will appear as “additional-rm” or the name you specified.
