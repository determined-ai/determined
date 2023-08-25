.. _basic-setup:

.. _install-cluster:

#############
 Basic Setup
#############

If you have not used Determined before, and you want to quickly set up a new training environment,
you are at the right place!

.. note::

   To set up a Determined training environment on-prem or on cloud, visit :ref:`Advanced Setup
   <advanced-setup>`.


**Prerequisites**

-  :ref:`Installation Requirements <system-requirements>`

*********************************
 Step 1 - Install Docker Desktop
*********************************

-  :ref:`Install Docker <install-docker>` on your machine.

*****************************
 Step 2 - Install Determined
*****************************

Install the Determined library and start a cluster locally.

-  Ensure you have Docker running and then run the following command:

.. code::

   pip install determined

   # If your machine has GPUs:
   det deploy local cluster-up

   # If your machine does not have GPUs:
   det deploy local cluster-up --no-gpu

.. include:: ../../_shared/note-pip-install-determined.txt

******************
 Step 3 - Sign In
******************

Once Determined is installed and Docker is running, you can sign in.

-  To do this, go to ``http://localhost:8080/``.
-  To sign in, accept the default determined username, leave the password empty, and then click
   **Sign In**.
