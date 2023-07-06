.. _saas-cloud-cluster:

###################
 Using Your Cluster
###################

.. meta::
   :description: Using your cloud cluster.


***************
 Authentication
***************

Your cluster can be accessed in one of two ways: through the Determined
Cloud web portal, or through the CLI.

In the web portal, click ``View Cluster`` to access the cluster’s web
interface.

To use the CLI, set the ``DET_MASTER`` environment variable to the
cluster’s URL (you can copy it using the clipboard button next to
``View Cluster``) and run ``det auth login``. This will open a page in
your browser to authenticate before returning to your terminal.


*************************
 Access Your Own Datasets
*************************

If you have training datasets in S3, you can easily access them from
your Determined Cloud clusters. The recommended method for doing this is
to configure your S3 buckets with a policy that grants access to your
Determined Cloud cluster. The required details and a template for this
policy can be accessed from the cluster menu. Click the ⋮ symbol, then
``Access Training Data``.

To access other storage, you may want to place the required credentials
in the environment variables for your experiment. However, be aware that
these will also be accessibly to your team.
