.. _topic_guide_mldes:

############################################################################################
 Deploy using HPE MLDES (Machine Learning Development Environment Software) Managed Service
############################################################################################

.. attention::

   Only Determined Enterprise Edition is available through MLDES.

.. note::

   Determined is referred to as "MLDE" in MLDES documentation and UI.

This section describes how to deploy a Determined cluster using MLDES. MLDES is a fully managed
service that abstracts the complexity of managing the underlying infrastructure and provides a
simple and easy-to-use interface to deploy, manage, and work with Determined clusters.

MLDES uses either an AWS or GCP account as a backend for Determined cluster resources. All you need
to create a new Determined cluster via MLDES is an AWS account with a few pre-requisite IAM roles,
or a GCP project ID. View the `MLDES Documentation <https://mldes.ext.hpe.com/docs/index.html>`_ for
more information. A simple guide to creating an MLDE cluster is described as follows:

#. Go to the `MLDES Console <https://mldes.ext.hpe.com>`_ and sign in with your HPE Passport
   account.
#. Create a new organization with a unique name and the necessary details.
#. Create a new Determined cluster by clicking on the "New MLDE Cluster" button.
#. Once the cluster is provisioned, open the Determined UI by clicking "Open" in the cluster details
   box.
#. Use the Determined UI to create and manage experiments, or open a Jupyter notebook to run your
   own code by clicking on the "Launch JupyterLab" button.
