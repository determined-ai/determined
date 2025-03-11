.. _topic_guide_mldes:

############################################################################################
 Deploy Using HPE Machine Learning Development Environment Software (MLDES) Managed Service
############################################################################################

.. attention::

   MLDE Managed Service has been discontinued. This page is now obsolete but is left here for
   historical reasons.

**********
 Overview
**********

You can deploy a Determined cluster using `MLDES <https://mldes.ext.hpe.com/docs/index.html>`__.
MLDES is a managed service that simplifies the infrastructure management complexities. It offers an
intuitive interface for deploying, managing, and utilizing Determined clusters.

***************
 Prerequisites
***************

MLDES supports both AWS and GCP as backends for Determined cluster resources. To start, ensure you
have an AWS account with the required IAM roles or a GCP project ID. For more information, visit the
`documentation <https://mldes.ext.hpe.com/docs/index.html>`__.

******************
 Deployment Guide
******************

Access the Console
==================

#. Visit the `MLDES console <https://mldes.ext.hpe.com>`_.
#. Sign in using your HPE Passport account.

Create an Organization
======================

#. Select **Organization** to open the Organizations panel and create a new organization.
#. Enter a unique name and select **Save**.

Deploy the Cluster
==================

#. Select **Clusters** to open the Clusters panel.
#. Select **New MLDE Cluster** to start creating a new cluster.
#. Enter a unique name.
#. Review the default cluster and resource pool configuration and make any necessary changes.
#. Select **Create Cluster**.

MLDE provisions the cluster and lets you know when it is ready.

Create and Manage Experiments
=============================

#. Once the cluster is ready, select **Open**.
#. To open a Jupyter notebook, **Launch JupyterLab**.
