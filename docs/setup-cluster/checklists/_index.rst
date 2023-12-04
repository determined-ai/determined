.. _advanced-setup-checklist:

#######################
 Advanced Installation
#######################

.. meta::
   :description: Follow this checklist when setting a Determined training environment on-prem or on cloud.

Using Determined requires a training environment. Your training environment can be a local
development machine, an on-premise GPU cluster, or cloud resources.

This checklist helps you get started setting up a new training environment for your organization.
After completing these steps, your users will be able to see and access your Determined cluster.

***************
 Prerequisites
***************

To complete the items in this checklist, ensure your system meets
:ref:`advanced-setup-requirements`.

About Offline Installations
===========================

-  If your master and compute nodes are offline, you'll need a local private registry that can
   satisfy necessary images (PostgreSQL + task container images).

-  You can install the Determined CLI package on your client machines and then take them offline
   again.

-  In addition, a local PyPi mirror for packages is highly recommended for installing packages from
   the internet in your task environments. See also:
   :ref:`advanced-setup-infrastructure-considerations`.

*******************
 Set Up PostgreSQL
*******************

Determined uses a PostgreSQL database to store experiment and trial metadata. Choose the
installation method that best fits your environment and requirements.

.. note::

   Kubernetes

   If you are using **Kubernetes**, you can skip this step. :ref:`Installing Determined on
   Kubernetes <determined-on-kubernetes>` uses the Determined Helm Chart which includes deployment
   of a PostgreSQL database.

.. note::

   Cloud Services

   -  :ref:`AWS <install-aws>`. The Determined CLI manages the process of provisioning an Amazon RDS
      instance for PostgreSQL.
   -  :ref:`GCP <install-gcp>`. The Determined CLI manages the setup of Google Cloud SQL instances
      for PostgreSQL.

.. tabs::

   .. tab::

      Linux

      :ref:`Installing Determined using Linux Packages <install-using-linux-packages-preliminary>`
      pulls in the official Docker image for PostgreSQL.

   .. tab::

      Docker

      :ref:`Setting Up a Docker PostgreSQL Environment <install-postgres-docker>`.

********************
 Install Determined
********************

Once PostgreSQL is set up, you'll install Determined. This includes deploying the Determined master,
configuring checkpoint storage, setting up resource pools, and configuring the cluster.

Deploy Determined Master
========================

To install Determined, decide if you want to deploy the Determined master on premises or on cloud.

.. tabs::

   .. tab::

      On Prem

      .. tabs::

         .. tab::

            Linux (Agent-Based)

            If the Determined agent is your compute resource, you'll install the Determined agent
            along with the Determined master. The preferred method for installing the Agent is to
            use Linux packages. The recommended alternative to Linux packages is Docker.

            To install the Determined master and agent on premises, you'll first need to meet the
            installation requirements:

            -  :ref:`Installation Requirements <requirements>`

            Once you've met the installation requirements, install the Determined Master and Agent:

            -  :ref:`Install Determined Using Linux Packages <install-using-linux-packages>`

            These instructions include editing the YAML configuration files for the master and each
            agent and for configuring and starting the cluster.

         .. tab::

            Docker (Agent-Based)

            If the Determined agent is your compute resource, you'll install the Determined agent
            along with the Determined master. The preferred method for installing the agent is to
            use Linux packages. The recommended alternative to Linux packages is Docker.

            To install the Determined master and agent on premises, you'll first need to meet the
            installation requirements:

            -  :ref:`Installation Requirements <requirements>`

            Once you've met the installation requirements, select one of the following options:

            -  :ref:`Install Determined Using Docker <install-using-docker>`

         .. tab::

            Kubernetes

            To install the Determined master on premises with Kubernetes, follow the steps below:

            -  :ref:`Deploy on Kubernetes <determined-on-kubernetes>`
            -  :ref:`Install Determined on Kubernetes <install-on-kubernetes>`

         .. tab::

            Slurm

            To install the Determined master on premises with Slurm, follow the steps below:

            -  :ref:`sysadmin-deploy-on-hpc`
            -  :ref:`install-on-slurm`

            Additional Resources

            -  :ref:`hpc-security-considerations`
            -  :ref:`slurm-image-config`

            Known Issues

            -  :ref:`known-hpc-issues`

   .. tab::

      On Cloud

      .. tabs::

         .. tab::

            Agent-Based

            To install the Determined master and agent on cloud, select one of the following
            options:

            -  :ref:`AWS <install-aws>`
            -  :ref:`GCP <install-gcp>`

            .. note::

               When using AWS or GCP, ``det CLI`` manages the installation of the Determined agent
               for you.

         .. tab::

            Kubernetes

            To install the Determined master on cloud using Kubernetes, start here:

            -  :ref:`Install on Kubernetes <install-on-kubernetes>`

            After completing the step above, select one of the following options:

            -  :ref:`setup-eks-cluster`
            -  :ref:`setup-gke-cluster`
            -  :ref:`setup-aks-cluster`

Configure Checkpoint Storage
============================

A checkpoint contains the architecture and weights of the model being trained. If
``checkpoint_storage`` is not specified, the experiment will default to the checkpoint storage
configured in the :ref:`master configuration <master-config-reference>`.

To learn more about configuring checkpoint storage, visit :ref:`checkpoint-storage`.

Configure Resource Pools
========================

When deploying the Determined master and compute resources (such as a Determined agent), you must
also configure :ref:`resource pools <resource-pools>`.

**How Resource Pools Work**

Both the Determined master and the compute resources, such as the Determined agents, come with their
individual configuration files. Among other things, these files define the resource pools and
specify how resources communicate and are allocated.

For instance, a Determined agent, which is a kind of compute resource, is part of a resource pool.
Its configuration file not only helps it communicate with the Determined master but also dictates
which resource pool it should connect to. By default, an agent will attempt to connect to the
"default" pool. However, if the "default" pool doesn't exist, the agent will remain unconnected.

**Setting Up an On-Prem Determined Agent**

For an on-prem Determined agent installation, the process involves the following steps:

-  Configure :ref:`resource pools <resource-pools>`. These resource pools enable the segregation of
   tasks based on their resource requirements.

-  Configure the agents to establish a connection to the Determined master. Then link the agents
   with their respective resource pools. For reference, visit :ref:`resource_pool
   <agent-resource-pool-reference>` under :ref:`agent-config-reference`.

Configure the Cluster
=====================

Once you have set up the necessary components for your environment, :ref:`configure the cluster
<cluster-configuration>`. When configuring your cluster, you'll need to keep the following resources
handy:

-  Common configuration reference: :ref:`common-configuration-options`
-  Master configuration reference: :ref:`master-config-reference`
-  Agent configuration reference: :ref:`agent-config-reference`

********************
 Configure Security
********************

After installing Determined, set up your :ref:`security <security-overview>` features.

.. attention::

   Security features, with the exception of TLS, are only available on Determined Enterprise Edition
   (Determined EE).

TLS
===

The use of :ref:`Transport Layer Security (TLS) <tls>` requires Determined EE and is highly
recommended.

.. tabs::

   .. tab::

      Master-Only TLS

      :ref:`Transport Layer Security (TLS) Master Configuration <tls-master>`

   .. tab::

      Mutual TLS

      `According to Wikipedia <https://en.wikipedia.org/wiki/Mutual_authentication>`_, Mutual
      authentication or two-way authentication refers to two parties authenticating each other at
      the same time in an authentication protocol. To require that agent connections be verified
      using mutual TLS, use `require_authentication` (for more information visit
      :ref:`master-config-reference`.

   .. tab::

      Agent-Based TLS

      In an agent-based installation, Determined is the resource manager. To set up TLS for Agents,
      visit :ref:`Transport Layer Security (TLS) Agents Configuration <tls-agents>`.

   .. tab::

      Kubernetes TLS

      :ref:`tls-on-kubernetes`

User Authentication (SSO)
=========================

Determined offers several options for user authentication:

+-------------------+----------------------------------------------------------------------------+
| Feature           | Description                                                                |
+===================+============================================================================+
| :ref:`oauth`      | Enable, list, and remove OAuth clients.                                    |
+-------------------+----------------------------------------------------------------------------+
| :ref:`oidc`       | Integrate OpenID Connect, with and Okta example.                           |
+-------------------+----------------------------------------------------------------------------+
| :ref:`saml`       | Integrate Security Assertion Markup Language (SAML) authentication to use  |
|                   | single sign-on (SSO) with your organizationidentity provider (IdP).        |
+-------------------+----------------------------------------------------------------------------+
| :ref:`scim`       | Integrate System for Cross-domain Identity Management (SCIM) for           |
|                   | administrators to easily and securely provision users and groups.          |
+-------------------+----------------------------------------------------------------------------+

.. note::

   For Kubernetes deployments, you modify the master-related configurations through the :ref:`helm
   <k8s-helm-reference>` chart.

Non-Root Containers
===================

You can enhance security and limit potential malicious activity by running containers as non-root
users. Determined allows you to :ref:`run tasks as specific agent users <run-as-user>` and :ref:`run
unprivileged tasks by default <run-unprivileged-tasks>`.

.. important::

   Red Hat® OpenShift® users should not follow these instructions for configuring non-root
   containers, as OpenShift's configuration conflicts with the approach described here.

To run containers as non-root users, you'll first need to set up your non-root user:

-  Choose a Determined user for configuration, preferably one who has not undergone the ``det user
   link-with-agent-user`` process and one you plan to eventually link with an agent user. If no
   suitable Determined user exists, consider creating a test user for this purpose, one which can be
   disabled afterwards.

-  Link this user to the actual username/UID and groupname/GID. One way to do this is to use the
   following command (you can also use the WebUI):

   .. code:: bash

      det user link-with-agent-user \
         --agent-user $THE_USER \
         --agent-uid $THE_UID \
         --agent-group $THE_GROUP \
         --agent-gid $THE_GID \
         $THE_DETERMINED_USER

-  Start a shell as the specified user:

   .. code:: bash

      det -u $THE_DETERMINED_USER shell start

-  In the shell, verify the username/UID and groupname/GID with ``id -a``.

-  After confirming the non-root containers are operational, you'll need to perform a test run of
   each training job you normally run as the modified Determined user. This ensures the training
   jobs run successfully without root privileges.

.. note::

   For Kubernetes deployments, configure the security context for running containers as a non-root
   user.

Configure Role-Based Access Control (RBAC)
==========================================

Consider configuring role-based access control (RBAC) before creating workspaces and projects. To
configure RBAC, visit :ref:`rbac`.

.. attention::

   RBAC is only available on Determined Enterprise Edition.

.. _advanced-setup-infrastructure-considerations:

*******************************
 Infrastructure Considerations
*******************************

When setting up Determined, you can adjust certain configurations for enhanced security and
performance. While these are particularly crucial for offline installations, they can also benefit
online installations by ensuring faster package retrieval and increased security.

Configure Local Docker Image Repositories
=========================================

Configuring local Docker image repositories can enhance security and optimize performance. Learn how
to configure local Docker image repositories in :ref:`Customizing Your Environment <custom-env>`.

Configure Local PyPi Mirrors
============================

It's recommended to consider configuring local PyPi mirrors for:

-  **Security**: An airgapped cluster, isolated from the public internet, mandates local mirrors for
   proper functionality. This also safeguards against potential vulnerabilities associated with
   fetching packages from external sources.

-  **Performance**: Local mirrors can substantially reduce the time taken to fetch packages,
   eliminating potential lags due to network issues or external server overloads.

********************
 Additional Options
********************

Create Workspaces and Projects
==============================

Determined lets you organize and control access to your experiments by team or department. To do
this, you can create :ref:`workspaces` based on your :ref:`rbac` groups. Once your workspaces are
set up, you can :ref:`bind resource pools to them <resource-pool-binding>`.

Set Up Monitoring Tools
=======================

To set up your monitoring tools, visit :ref:`configure-prometheus-grafana`.

Configure Infiniband
====================

You may choose to configure :ref:`InfiniBand <infiniband>` when connecting multiple data streams in
a single connection.

****************
 Set Up Clients
****************

You can :ref:`set up clients <setup-clients>` for interacting with the Determined master through the
CLI to provide users with efficient access for task execution without having to go through the
WebUI.

*****************
 Test Your Setup
*****************

Test your setup to ensure it is functioning correctly.

.. tabs::

   .. tab::

      Run a Single CPU/GPU Training Job

      Test that you can run a single CPU/GPU training job.

      #. Download the :download:`mnist_pytorch.tgz </examples/mnist_pytorch.tgz>` file to a local
         directory.

      #. Open a terminal window, extract the files, and ``cd`` into the ``mnist_pytorch`` directory:

         .. code:: bash

            tar xzvf mnist_pytorch.tgz
            cd mnist_pytorch

      #. In the ``mnist_pytorch`` directory, create an experiment specifying the ``const.yaml``
         configuration file:

         .. code:: bash

            det experiment create const.yaml .

         You should receive confirmation that the experiment is created:

         .. code:: console

            Preparing files (.../mnist_pytorch) to send to master... 8.6KB and 7 files
            Created experiment 1

      #. Enter the cluster address in the browser address bar to view experiment progress in the
         WebUI.

         You should be able to see your experiment ID and its status.

   .. tab::

      Run a Distributed Training Job

      Test that you can run a remote distributed training job.

      The ``distributed.yaml`` configuration file for this step is the same as the ``const.yaml``
      file in the previous step, except that a ``resources.slots_per_trial`` field is defined and
      set to a value of ``8``:

      .. code:: yaml

         resources:
            slots_per_trial: 8

      This is the number of available GPU resources. The ``slots_per_trial`` value must be divisible
      by the number of GPUs per machine. You can change the value to match your hardware
      configuration.

      #. To connect to a Determined master running on a remote instance, set the remote IP address
         and port number in the ``DET_MASTER`` environment variable:

         .. code:: bash

            export DET_MASTER=<ipAddress>:8080

      #. Create and run the experiment:

         .. code:: bash

            det experiment create distributed.yaml .

         You can also use the ``-m`` option to specify a remote master IP address:

         .. code:: bash

            det -m http://<ipAddress>:8080 experiment create distributed.yaml .

      #. To view the WebUI dashboard, enter the cluster address in your browser address bar, accept
         ``determined`` as the default username, and click **Sign In**. A password is not required.

      #. Click the **Experiment** name to view the experiment’s trial display.

   .. tab::

      Verify Users Can Access the Cluster

      Test that your users can access the cluster.

      To view the WebUI dashboard, enter the cluster address in the browser address bar, accept the
      default username of ``determined``, and click **Sign In**. A password is not required.

************
 Next Steps
************

Congratulations! You have set up your Determined environment! Your users should be able to see and
connect to the Determined master.

.. toctree::
   :hidden:
   :glob:

   ./*
