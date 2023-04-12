.. _install-gcp:

####################
 Install Determined
####################

This document describes how to deploy a Determined cluster on Google Cloud Platform (GCP). The ``det
deploy`` tool makes it easy to create and deploy these resources in GCP. The ``det deploy`` tool
uses `Terraform <https://learn.hashicorp.com/terraform/getting-started/install.html>`__ to
automatically deploy and configure a Determined cluster in GCP. Alternatively, if you already have a
process for setting up infrastructure with Terraform, you can use our `Terraform modules
<https://github.com/determined-ai/determined/tree/master/harness/determined/deploy/gcp/terraform>`__
rather than ``det deploy``.

For more information about using Determined on GCP, see the :ref:`topic_guide_gcp` topic guide.

**************
 Requirements
**************

Project
=======

To get started on GCP, you will need to create a `project
<https://cloud.google.com/appengine/docs/standard/nodejs/building-app/creating-project>`__.

The following GCP APIs must be enabled on your GCP project:

-  `Cloud Filestore API <https://console.cloud.google.com/apis/library/file.googleapis.com>`__
-  `Cloud Resource Manager API
   <https://console.developers.google.com/apis/library/cloudresourcemanager.googleapis.com>`__
-  `Cloud SQL Admin API
   <https://console.developers.google.com/apis/library/sqladmin.googleapis.com>`__
-  `IAM API <https://console.developers.google.com/apis/api/iam.googleapis.com/overview>`__
-  `Service Networking API
   <https://console.cloud.google.com/apis/library/servicenetworking.googleapis.com>`__
-  `Cloud Logging API <https://console.cloud.google.com/apis/api/logging.googleapis.com/overview>`__

Credentials
===========

The ``det deploy`` tool requires credentials in order to create resources in GCP. There are two ways
to provide these credentials:

-  Use `gcloud <https://cloud.google.com/sdk/docs/downloads-interactive#installation_options>`__ to
   authenticate your user account:

   .. code::

      gcloud auth application-default login

   This command will open a login page in your browser where you can sign in to the Google account
   that has access to your project. Ensure your user account has ``Owner`` access to the project you
   want to deploy your cluster in.

-  Use :ref:`service account credentials <gcp-service-account-credentials>`.

Resource Quotas
===============

The default `GCP Resource Quotas <https://cloud.google.com/compute/quotas>`__ for GPUs are
relatively low; you may wish to request a quota increase.

.. _gcp-install:

*********
 Install
*********

#. Install `Terraform <https://learn.hashicorp.com/terraform/getting-started/install.html>`__.

#. Install ``determined`` using ``pip``:

   .. code::

      pip install determined

******************
 Deploy a Cluster
******************

We recommend creating a new directory and running the commands below inside that directory.

.. note::

   The deployment process will create Terraform state and variables files in the directory where it
   is run. The state file keeps track of deployed resources and their state and is used to update or
   delete the cluster in the future. The variables files includes all Terraform variables used for
   deployment (e.g., service account keypath, cluster ID, GCP region and zone).

   Any future update or deletion commands should be run inside the same directory so ``det deploy``
   can read the state and variables files. If either of these files is deleted, it will be difficult
   to manage the deployment afterward. Storing these files in a safe location is strongly
   recommended.

To deploy the cluster, run:

.. code::

   det deploy gcp up --cluster-id CLUSTER_ID --project-id PROJECT_ID

``CLUSTER_ID`` is an arbitrary unique ID for the new cluster. We recommend choosing a cluster ID
that is memorable and helps identify what the cluster is being used for.

The deployment process may take 5-10 minutes. When it completes, summary information about the newly
deployed cluster will be printed, including the URL of the Determined master.

Required Arguments:
===================

.. list-table::
   :widths: 25 50 25
   :header-rows: 1

   -  -  Argument
      -  Description
      -  Default Value

   -  -  ``--cluster-id``
      -  A string appended to resources to uniquely identify the cluster.
      -  *required*

   -  -  ``--project-id``
      -  The project to deploy the cluster in.
      -  *required*

Optional Arguments:
===================

.. list-table::
   :widths: 25 50 25
   :header-rows: 1

   -  -  Argument
      -  Description
      -  Default Value

   -  -  ``--keypath``
      -  The path to the service account JSON key file if using a service account. Including this
         flag will supersede default Google Cloud user credentials.
      -  Not set

   -  -  ``--preemptible``
      -  Whether to use preemptible dynamic agent instances.
      -  False

   -  -  ``--gpu-type``

      -  The type of GPU to use for the agent instances. Ensure ``gpu_type`` is available in your
         selected ``region`` and ``zone`` by referring to the `GPUs on Compute Engine
         <https://cloud.google.com/compute/docs/gpus>`__ page.

      -  nvidia-tesla-t4

   -  -  ``--gpu-num``

      -  The number of GPUs on each agent instance. Between 0 and 8 (more GPUs require a more
         powerful ``agent-instance-type``). Refer to the `GPUs on Compute Engine
         <https://cloud.google.com/compute/docs/gpus>`__ page for specific GCP requirements. Can be
         set to 0 for CPU-based training.

      -  8

   -  -  ``--max-dynamic-agents``
      -  Maximum number of dynamic agent instances at one time.
      -  5

   -  -  ``--max-aux-containers-per-agent``
      -  The maximum number of containers running for agents in the auxiliary resource pool.
      -  100

   -  -  ``--max-idle-agent-period``
      -  The length of time to wait before idle dynamic agents will be automatically terminated.
      -  10m

   -  -  ``--network``
      -  The network to create (ensure there isn't a network with the same name already in the
         project, otherwise the deployment will fail).
      -  det-default-``cluster-id``

   -  -  ``--region``
      -  The region to deploy the cluster in.
      -  us-west1

   -  -  ``--zone``
      -  The zone to deploy the cluster in.
      -  ``region``-b

   -  -  ``--master-instance-type``
      -  Instance type to use for the master instance.
      -  n1-standard-2

   -  -  ``--aux-agent-instance-type``
      -  Instance type to use for the agent instances in the auxiliary resource pool.
      -  n1-standard-4

   -  -  ``--compute-agent-instance-type``
      -  Instance type to use for the agent instances in the compute resource pool.
      -  n1-standard-32

   -  -  ``--min-cpu-platform-master``
      -  Minimum CPU platform for the master instance.
      -  Intel Skylake

   -  -  ``--min-cpu-platform-agent``

      -  Minimum CPU platform for the agent instances. Ensure the platform is compatible with your
         selected ``gpu-type`` and available in your selected ``region`` and ``zone`` by referring
         to the `GPUs on Compute Engine <https://cloud.google.com/compute/docs/gpus>`__ page.

      -  Intel Broadwell

   -  -  ``--local-state-path``
      -  Directory used to store cluster metadata. The same directory cannot be used for multiple
         clusters at the same time.
      -  Current working directory

   -  -  ``--master-config-template-path``
      -  Path to the custom ``master.yaml`` template. Default template can be obtained using ``det
         deploy gcp dump-master-config-template``.
      -  Not set

The following ``gcloud`` commands will help to validate your configuration, including resource
availability in your desired region and zone:

.. code::

   # Validate that the GCP Project ID exists.
   gcloud projects list

   # Verify that the environment_image is listed.
   gcloud compute images list --filter=name:<environment_image>

   # Check that a zone is available in the configured region.
   gcloud compute zones list --filter=region:<region>

   # List the available machine types (for master_machine_type and agent_machine_type) in the configured zone.
   gcloud compute machine-types list --filter=zone:<zone>

   # List the valid gpu_type values for the configured zone.
   gcloud compute accelerator-types list --filter=zone:<zone>

******************
 Update a Cluster
******************

If you need to make changes to your cluster, you can rerun ``det deploy gcp up [args]`` in the same
directory and your cluster will be updated. The ``det deploy`` tool will only replace resources that
need to be replaced based on the changes you've made in the updated execution.

.. note::

   If you'd like to change the ``region`` of a deployment after it has already been deployed, we
   recommend deleting the cluster first, then redeploying the cluster with the new ``region``.

*******************
 Destroy a Cluster
*******************

To bring down the cluster, run the following in the same directory where you ran ``det deploy gcp
up``:

.. code::

   det deploy gcp down

``det deploy`` will use the ``.tfstate`` and ``terraform.tfvars.json`` files in the current
directory to determine which resources to destroy. If you deployed with a service account JSON key
file, the same credentials file will be used for deprovisioning. Otherwise, default Google Cloud
credentials are used.

.. _gcp-master-yaml-template:

******************************
 Custom master.yaml templates
******************************

Similarly to a corresponding :ref:`AWS feature <aws-master-yaml-template>`, advanced users who
require a deep customization of master settings (i.e., the ``master.yaml`` config file) can use the
``master.yaml`` templating feature. Since ``det deploy gcp`` fills in plenty of
infrastructure-related values such as subnetwork ids or boot disk images, we provide a simplified
templating solution, similar to :ref:`helm charts in kubernetes <install-on-kubernetes>`. Template
language is based on golang templates, and includes ``sprig`` helper library and ``toYaml``
serialization helper.

Example workflow:

#. Get the default template using

   .. code::

      det deploy gcp dump-master-config-template > /path/to/master.yaml.tmpl

#. Customize the template as you see fit by editing it in any text editor. For example, let's say a
   user wants to utilize (default) 4-GPU instances for the default compute pool, but they also often
   run single-GPU notebook jobs, for which a single-GPU instance would be perfect. So, you want to
   add a third pool ``compute-pool-solo`` with a customized instance type.

   Start with the default template, and find the ``resource_pools`` section:

   .. code:: yaml

      resource_pools:
      - pool_name: aux-pool
        max_aux_containers_per_agent: {{ .resource_pools.pools.aux_pool.max_aux_containers_per_agent }}
        provider:
          instance_type:
            {{- toYaml .resource_pools.pools.aux_pool.instance_type | nindent 8 }}
          {{- toYaml .resource_pools.gcp | nindent 6}}

      - pool_name: compute-pool
        max_aux_containers_per_agent: 0
        provider:
          instance_type:
            {{- toYaml .resource_pools.pools.compute_pool.instance_type | nindent 8 }}
          cpu_slots_allowed: true
          {{- toYaml .resource_pools.gcp | nindent 6}}:

   Then, append a new section:

   .. code:: yaml

      - pool_name: compute-pool-solo
        max_aux_containers_per_agent: 0
        provider:
          instance_type:
             machine_type: n1-standard-4
             gpu_type: nvidia-tesla-t4
             gpu_num: 1
             preemptible: false
       {{- toYaml .resource_pools.gcp | nindent 6}}

#. Use the new template:

   .. code::

      det deploy gcp <ALL PREVIOUSLY USED FLAGS> --master-config-template-path /path/to/edited/master.yaml.tmpl

#. All set! Check the `Cluster` page in WebUI to ensure your cluster has 3 resource pools. In case
   of errors, ssh to the master instance as instructed by ``det deploy gcp`` output, and check
   ``sudo journalctl -u google-startup-scripts.service``, ``/var/log/cloud-init-output.log``, or
   ``sudo docker logs determined-master``.

.. _gcp-service-account-credentials:

*****************************
 Service Account Credentials
*****************************

For more security controls, you can create a `service account
<https://cloud.google.com/docs/authentication/getting-started>`__ or select an existing service
account from the `service account key page in the Google Cloud Console
<https://console.cloud.google.com/apis/credentials/serviceaccountkey>`__ and ensure it has the
following IAM roles:

-  Cloud Filestore Editor
-  Cloud SQL Admin
-  Compute Admin
-  Compute Network Admin
-  Security Admin
-  Service Account Admin
-  Service Account User
-  Service Networking Admin
-  Storage Admin

Roles provide the service account permissions to create specific resources in your project. You can
add roles to service accounts following this `guide
<https://cloud.google.com/iam/docs/granting-roles-to-service-accounts>`__.

Once you have a service account with the appropriate roles, go to the `service account key page in
the Google Cloud Console <https://console.cloud.google.com/apis/credentials/serviceaccountkey>`__
and create a JSON key file. Save it to a location you'll remember; we'll refer to the path to this
key file as the ``keypath``, which is an optional argument you can supply when using ``det deploy``.
Once you have the ``keypath`` you can use it to deploy a GCP cluster by continuing the
:ref:`installation <gcp-install>` section.

.. _gcp-det-deploy-a100:

************************************
 Run Determined on NVIDIA A100 GPUs
************************************

Determined makes it possible to try out your models on latest NVIDIA A100 GPUs; however, there are a
few considerations:

-  A100s may not be available in your default GCP region and zone, and you may need to specify a
   different one explicitly. `See more on GPU availablity
   <https://cloud.google.com/compute/docs/gpus/gpu-regions-zones>`__.

-  Make sure you have sufficient resource quota for A100s in your target region and zone. `See more
   on quotas <https://cloud.google.com/compute/quotas>`__.

-  Adjust maximum number of instances and to be within your quota using ``--max-dynamic-agents
   NUMBER``.

This command line will spin up a cluster of up to 2 A100s in the ``us-central1-c`` zone:

.. code::

   det deploy gcp up --cluster-id CLUSTER_ID --project-id PROJECT_ID \
      --max-dynamic-agents 2 \
      --compute-agent-instance-type a2-highgpu-1g --gpu-num 1 \
      --gpu-type nvidia-tesla-a100 \
      --region us-central1 --zone us-central1-c \
      --gpu-env-image determinedai/environments-dev:cuda-11.3-pytorch-1.12-tf-2.8-gpu-0.21.2 \
      --cpu-env-image determinedai/environments-dev:py-3.8-pytorch-1.12-tf-2.8-cpu-0.21.2
