.. _saas-cloud-get-started:

##################################
 Get Started with Determined Cloud
##################################

.. meta::
   :description: Learn how to get started with managed cloud infrastructure.

Determined Cloud provides access to managed cloud infrastructure for
training AI models at scale. You get access to AI clusters running the
`HPE Machine Learning Development
Environment <https://www.hpe.com/us/en/solutions/artificial-intelligence/machine-learning-development-system.html>`__
without having to provision or configure the hardware yourself.
Determined Cloud clusters scale automatically with demand, so you only
pay for what you use. This is the best way to start training
deep-learning models at scale and collaborate with your team.

********************
 Create Your Cluster
********************

To get started, let’s create a cluster to train a model on. Click the
``New Cluster`` button. Choose a name for your cluster, and select the
``Standard`` configuration. You may also choose the ``Pro``
configuration which is configured with more powerful GPUs. You may also
customize the configuration in the ``Advanced`` menu, and modify this
configuration later.

********************
 Train Your Model
********************

While your cluster is launching, install the Determined command-line
tool. You must already have
`Python <https://www.python.org/downloads/>`__ installed.

.. code:: bash

   pip install determined

Get the cluster URL (you can copy it using the clipboard button next to
``View Cluster``) and set the ``DET_MASTER`` environment variable. You
may want to set this permanently for your shell (e.g. add it to
``.bashrc``, etc.).

.. code:: bash

   export DET_MASTER=<master ip>

Once the cluster is running, you can tell the CLI to log you into the
cluster through your browser.

.. code:: bash

   det auth login

Now you can download an example model and train it using your cluster.

.. code:: bash

   curl -O https://docs.determined.ai/latest/_downloads/61c6df286ba829cb9730a0407275ce50/mnist_pytorch.tgz
   tar xzf mnist_pytorch.tgz
   cd mnist_pytorch
   det experiment create const.yaml .

The experiment, its progress and metrics can now be found in the
cluster’s web UI.

You can replace ``const.yaml`` with ``distributed.yaml`` to demonstrate
distributed training over multiple GPUs, and ``adaptive.yaml`` to
demonstrate automatic hyperparameter optimization.

You can visit the ``Docs`` tab in your cluster for complete
documentation for model developers on your specific version.

Working with Checkpoints
========================

Please consult the `Determined
docs <https://docs.determined.ai/latest/training/model-management/checkpoints.html>`__
for full details on working with checkpoints.

Your experiments will likely generate checkpoints. You can use the
command line below to view all checkpoints associated with an
experiment:

.. code:: bash

   det experiment list-checkpoints <experiment-id>

To download them on Determined Cloud, use this command line:

.. code:: bash

   det checkpoint download --mode=master <checkpoint UUID>

The option ``--mode=master`` explicitly specifies the download is
proxied through the cluster master, which is the only supported download
mode on Determined Cloud.

For consistency with the open-source Determined, you can also omit
``--mode=master``:

.. code:: bash

   det checkpoint download <checkpoint UUID>

**Caveat**: When ``--mode`` is not specified, it defaults to ``auto``.
The CLI will first attempt to download checkpoints from its storage
directly and then will fail over to proxied download through the cluster
master. This command line is easier to remember but has a small
overhead. In some occasions, it might fail to automatically switch to
proxied download.

Add Your Team
=============

Back in the Determined Cloud web portal, click the ``Members`` tab. Your
user is currently an ``admin`` of the organization and has all
permissions. This includes the ability to add other team members. Click
``New Member`` and enter their email address. Send them a link, and
they’ll be able to log in and collaborate with you!
