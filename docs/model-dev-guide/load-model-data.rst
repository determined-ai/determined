.. _load-model-data:

.. _prepare-data:

##############
 Prepare Data
##############

Data plays a fundamental role in machine learning model development. The best way to load data into
your ML models depends on several factors, including whether you are running on-premise or in the
cloud, the size of your data sets, and your security requirements. Accordingly, Determined supports
a variety of methods for accessing data.

The easiest way to include data in your experiment is to add the data to the same directory as your
model code. When you create a new experiment, all files in your model code directory will be
packaged and uploaded to the Determined cluster, assuming the package is smaller than 96MB.

If your dataset is larger than 96MB, you can use a :ref:`Startup Hook <startup-hooks>` shell script
to download the dataset prior to training.

This document also introduces production data source options, such as Object Store ( `Amazon S3
<https://aws.amazon.com/s3/>`__, `Google Cloud Storage <https://cloud.google.com/storage>`__) or
Distributed File Systems (`NFS <https://en.wikipedia.org/wiki/Network_File_System>`__, `Ceph
<https://ceph.io/>`__) if the above approaches do not apply to you.

*******************************
 Including Data With Your Code
*******************************

The data set can be uploaded as part of the :ref:`experiment <experiments>` directory, which usually
include your training API implementation. The size of this directory must not exceed 96MB, so this
method is only appropriate when the size of the data set is small.

For example, the experiment directory below contains the model definition, an experiment
configuration file, and a CSV data file. All three files are small and hence the total size of the
directory is much smaller than the 96MB limit:

.. code::

   .
   ├── const.yaml (0.3 KB)
   ├── data.csv (5 KB)
   └── model_def.py (4.1 KB)

The data can be submitted along with the model definition using the command:

.. code::

   det create experiment const.yaml .

Determined injects the contents of the experiment directory into each trial container that is
launched for the experiment. Any file in that directory can then be accessed by your model code,
e.g., by relative path (the model definition directory is the initial working directory for each
trial container).

For example, the code below uses `Pandas <https://pandas.pydata.org/>`__ to load ``data.csv`` into a
`DataFrame <https://pandas.pydata.org/pandas-docs/stable/reference/api/pandas.DataFrame.html>`__:

.. code:: python

   df = pandas.read_csv("data.csv")

*************************
 Distributed File System
*************************

Another way to store data is to use a distributed file system, which enables a cluster of machines
to access a shared data set via the familiar POSIX file system interface. Amazon's `Elastic File
System <https://aws.amazon.com/efs/>`__ and Google's `Cloud Filestore
<https://cloud.google.com/filestore>`__ are examples of distributed file systems that are available
in cloud environments. For on-premise deployments, popular distributed file systems include `Ceph
<https://ceph.io/>`__, `GlusterFS <https://www.gluster.org/>`__, and `NFS
<https://en.wikipedia.org/wiki/Network_File_System>`__.

To access data on a distributed file system, you should first ensure that the file system is mounted
at the same mount point on every Determined agent. For cloud deployments, this can be done by
configuring ``provisioner.startup_script`` in ``master.yaml`` to point to a script that mounts the
distributed file system. An example of how to do this on GCP can be :ref:`found here
<gcp-attach-disk>`.

Next, you will need to ensure the file system is accessible to each trial container. This can be
done by configuring a bind mount in the :ref:`experiment configuration file
<experiment-config-reference>`. Each bind mount consists of a ``host_path`` and a
``container_path``; the host path specifies the absolute path where the distributed file system has
been mounted on the agent, while the container path specifies the path within the container's file
system where the distributed file system will be accessible.

To avoid confusion, you may wish to set the ``container_path`` to be equal to the ``host_path``. You
may also want to set ``read_only`` to ``true`` for each bind mount, to ensure that data sets are not
modified by training code.

The following example assumes a Determined cluster is configured with a distributed file system
mounted at ``/mnt/data`` on each agent. To access data on this file system, we use an experiment
configuration file as follows:

.. code:: yaml

   bind_mounts:
     - host_path: /mnt/data
       container_path: /mnt/data
       read_only: true

Our model definition code can then access data in the ``/mnt/data`` directory as follows:

.. code:: python

   def build_training_data_loader(self):
       return make_data_loader(data_path="/mnt/data/training", ...)


   def build_validation_data_loader(self):
       return make_data_loader(data_path="/mnt/data/validation", ...)

****************
 Object Storage
****************

Object stores manage data as a collection of key-value pairs. Object storage is particularly popular
in cloud environments -- for example, Amazon's `Simple Storage Service
<https://aws.amazon.com/s3/>`__ (S3) and `Google Cloud Storage <https://cloud.google.com/storage>`__
(GCS) are both object stores. When running Determined in the cloud, it is highly recommended that
you store your data using the same cloud provider being used for the Determined cluster itself.

Unless you are accessing a publicly available data set, you will need to ensure that Determined
trial containers can access data in the object storage service you are using. This can be done by
configuring a :ref:`custom environment <custom-env>` with the appropriate credentials. When using
:ref:`Dynamic Agents on GCP <dynamic-agents-gcp>`, a system administrator will need to configure a
valid :ref:`service account <cluster-configuration>` with read credentials. When using :ref:`Dynamic
Agents on AWS <dynamic-agents-aws>`, the system administrator will need to configure an
:ref:`iam_instance_profile_arn <cluster-configuration>` with read credentials.

Once security access has been configured, we can use open-source libraries such as `boto3
<https://aws.amazon.com/sdk-for-python/>`__ or `gcsfs <https://gcsfs.readthedocs.io/en/latest/>`__
to access data from object storage. The simplest way to do this is for your model definition code to
download the entire data set whenever a trial container starts up.

Downloading from Object Storage
===============================

The example below demonstrates how to download data from S3 using ``boto``. The S3 bucket name is
specified in the experiment config file (using a field named ``data.bucket``). The
``download_directory`` variable defines where data that is downloaded from S3 will be stored. Note
that we include :func:`self.context.distributed.get_rank()
<determined._core._distributed.DistributedContext.get_rank>` in the name of this directory: when
doing distributed training, multiple processes might be downloading data concurrently (one process
per GPU), so embedding the rank in the directory name ensures that these processes do not conflict
with one another.

.. include:: ../_shared/note-dtrain-learn-more.txt

Once the download directory has been created, ``s3.download_file(s3_bucket, data_file, filepath)``
fetches the file from S3 and stores it at the specified location. The data can then be accessed in
the ``download_directory``.

.. code:: python

   import boto3
   import os


   def download_data_from_s3(self):
       s3_bucket = self.context.get_data_config()["bucket"]
       download_directory = f"/tmp/data-rank{self.context.distributed.get_rank()}"
       data_file = "data.csv"

       s3 = boto3.client("s3")
       os.makedirs(download_directory, exist_ok=True)
       filepath = os.path.join(download_directory, data_file)
       if not os.path.exists(filepath):
           s3.download_file(s3_bucket, data_file, filepath)
       return download_directory

To use this in your trial class, start by calling ``download_data_from_s3`` in the trial's
``__init__`` function. Next, implement the ``build_training_data_loader`` and
``build_validation_data_loader`` functions to load the training and validation data sets,
respectively, from the downloaded data.

Streaming from Object Storage
=============================

Rather than downloading the entire training data set from object storage during trial startup,
another way to load data is to *stream* batches of data from the training and validation sets as
needed. This has several advantages:

-  It avoids downloading the entire data set during trial startup, allowing training tasks to start
   more quickly.

-  If a container doesn't need to access the entire data set, streaming can result in downloading
   less data. For example, when doing hyperparameter searches, many trials can often be terminated
   after having been trained for less than a full epoch.

-  If the data set is extremely large, streaming can avoid the need to store the entire data set on
   disk.

-  Streaming can allow model training and data downloading to happen in parallel, improving
   performance.

To perform streaming data loading, the data must be stored in a format that allows efficient random
access, so that the model code can fetch a specific batch of training or validation data. One way to
do this is to store each batch of data as a separate object in the object store. Alternatively, if
the data set consists of fixed-size records, you can use a single object and then read the
appropriate byte range from it.

To stream data, a custom ``torch.utils.data.Dataset`` or ``tf.keras.utils.Sequence`` object is
required, depending on whether you are using PyTorch or TensorFlow Keras, respectively. These
classes require a ``__getitem__`` method that is passed an index and returns the associated batch or
record of data. When streaming data, the implementation of ``__getitem__`` should fetch the required
data from the object store.

The code below demonstrates a custom ``tf.keras.utils.Sequence`` class that streams data from Amazon
S3. In the ``__getitem__`` method, ``boto3`` is used to fetch the data based on the provided bucket
and key.

.. code:: python

   import boto3


   class ObjectStorageSequence(tf.keras.utils.Sequence):
       ...

       def __init__(self):
           self.s3_client = boto3.client("s3")

       def __getitem__(self, idx):
           bucket, key = get_s3_loc_for_batch(idx)
           blob_data = self.s3_client.get_object(Bucket=bucket, Key=key)["Body"].read()
           return data_to_batch(blob_data)
