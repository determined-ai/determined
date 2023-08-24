.. _basic-setup:

.. _install-cluster:

#############
 Basic Setup
#############

If you have not used Determined before, and you want to quickly set up a new training environment,
you are at the right place!

.. note::

   To set up a Determined training environment on-prem or on cloud, visit the :ref:`Advanced Setup
   <advanced-setup>`.

**Prerequisites**

-  :doc:`Installation Requirements </setup-cluster/deploy-cluster/on-prem/requirements>`

*********************************
 Step 1 - Install Docker Desktop
*********************************

:ref:`Install Docker <install-docker>` on your machine.

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

.. _firewall-rules:

************************************
 Transferring the Context Directory
************************************

You can use the ``-c <directory>`` option to transfer files from a directory on your local machine,
called the context directory, to the container. The context directory contents are placed in the
container working directory before the command or shell run. Files in the context can be accessed
using relative paths.

.. code::

   $ mkdir context
   $ echo 'print("hello world")' > context/run.py
   $ det cmd run -c context python run.py

The total size of the files in the context directory must be less than 95 MB. Larger files, such as
datasets, must be mounted into the container, downloaded after the container starts, or included in
a :ref:`custom Docker image <custom-docker-images>`.
