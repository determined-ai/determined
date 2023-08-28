.. _basic-setup:

.. _install-cluster:

#############
 Basic Setup
#############

If you have not used Determined before, and you want to quickly set up a new training environment,
you are at the right place!

.. note::

   For more advanced installations, visit :ref:`Advanced Setup <advanced-setup>`.

**Prerequisites**

-  Your system must meet the software and hardware requirements described in the :ref:`Installation
   Requirements <system-requirements>`.

*************************
 Step 1 - Install Docker
*************************

-  :ref:`Install Docker <install-docker>` on your machine.

*****************************
 Step 2 - Install Determined
*****************************

Install the Determined library and start a cluster locally.

-  Ensure you have Docker running and then run the following commands:

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

-  Go to ``http://localhost:8080/``.
-  Accept the default determined username, leave the password empty, and then click **Sign In**.
