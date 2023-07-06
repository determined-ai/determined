.. _deploy-gcp-cloud:

###########################
 Bring Your Own Cloud: GCP
###########################

.. meta::
   :description: Steps for integrating your cloud provider account with Determined.

*****
 GCP
*****

To grant Determined Cloud access to your GCP account, you will need to connect the Determined Cloud
Service account to your GCP account. You can do this with the Google Cloud management console or
``gcloud`` CLI.

Connecting with the Google Cloud management console
===================================================

#. Navigate to the IAM page within Google Cloud console. Click `here
   <https://console.cloud.google.com/iam-admin/iam?walkthrough_id=iam--quickstart>`__ for navigation
   guidance to the IAM page

#. Once on the IAM page, click the ``Grant Access`` button

#. Enter ``saas-x-acct@determined-ai.iam.gserviceaccount.com`` as the principal

#. From the ``Select a role`` drop-down menu, in the ``Quick access`` section, select ``Basic`` and
   then ``Editor``

#. Click ``Save``

Connecting with the gcloud CLI
==============================

#. Install the gcloud CLI if not already installed. See the `install
   <https://cloud.google.com/sdk/docs/install>`__ page for more information on how to do so

#. Login to the project you want Determined Cloud deployed to. More information on CLI login can be
   found on the `gcloud auth login <https://cloud.google.com/sdk/gcloud/reference/auth/login>`__
   page

#. Run

.. code::

   gcloud projects add-iam-policy-binding project-name --member="serviceAccount:saas-x-acct@determined-ai.iam.gserviceaccount.com" --role="roles/editor"

Make sure to change ``project-name`` to the name of the project you want Determined Cloud deployed
to.

Required Roles
==============

TODO: Right now we are just setting ``roles/editor`` here. Go back in and set the required roles in
this doc once we know what they will be.
