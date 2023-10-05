:orphan:

.. _det-pach-cat-dog:

#######################################
 Integrating Determined with Pachyderm
#######################################

.. meta::
   :description: Learn how to integrate Determined with Pachyderm for efficient ML workflows. This user guide walks you through the setup and verification.

In this guide, we'll help you get Determined and Pachyderm up and running together. Once integrated, these platforms can streamline your ML pipelines.

************
 Objectives
************

By following these instructions, you will:

-  Install Determined on your system.
-  Install Pachyderm locally.
-  Confirm that both Determined and Pachyderm are installed and connected.

After completing the steps in this user guide, you will have a foundation to build, train, and deploy ML models using both platforms.

***************
 Prerequisites
***************

**Required**

-  A compatible operating system for Determined and Pachyderm.
-  Necessary system permissions for software installation.

**Recommended**

-  Familiarity with basic ML concepts.

*************************
 Install Determined
*************************

To integrate Determined with Pachyderm, you first need to install Determined. Follow the installation guide provided:

- `Determined Installation Guide <https://docs.determined.ai/latest/setup-cluster/basic.html>`_

**********************
 Install Pachyderm
**********************

Once Determined is set up, proceed to install Pachyderm. Use the instructions in the link below:

- `Pachyderm Local Deployment Guide <https://docs.pachyderm.com/latest/set-up/local-deploy/>`_

*************************************
 Confirm Installation and Connection
*************************************

After installing both tools, it's essential to verify that they're installed correctly and can communicate with each other.

1. Check the Determined version and configuration:

.. code:: bash

   !det version

You should see a configuration summary with details like the version number, master address, and more.

2. Verify the Pachyderm version:

.. code:: bash

   !pachctl version

Ensure that the reported versions for `pachctl` and `pachd` match the versions you've installed.

**************************************************
 Create a Pachyderm Project for Batch Inferencing
**************************************************

To streamline batch inferencing, we'll encapsulate our training and inferencing processes within a Pachyderm project.

- Start by creating a new Pachyderm project:

.. code:: bash

   !pachctl create project batch-inference-1

- Next, update your Pachyderm configuration to set the context to the project you've just created:

.. code:: bash

   !pachctl config update context --project batch-inference-1

By setting up a dedicated project, you ensure that all related repos and pipelines are organized cohesively. This encapsulation makes it easier to manage batch inferencing workflows in the future.

********************************
 Create Repos for Training Data
********************************

To manage your training data effectively, you'll need to create repositories in Pachyderm. The training data is typically split in an 80:20 ratio, where 80% is used for training and 20% is used to validate the model.

To achieve this, run the following commands:

.. code:: bash

   !pachctl create repo test
   !pachctl create repo train
   !pachctl list repo

The expected output should be:

+-------------------+-------+----------------+-----------------+-------------+
| PROJECT           | NAME  | CREATED        | SIZE (MASTER)   | DESCRIPTION |
+===================+=======+================+=================+=============+
| batch-inference-1 | train | 3 seconds ago  | ≤ 0B            |             |
+-------------------+-------+----------------+-----------------+-------------+
| batch-inference-1 | test  | 6 seconds ago  | ≤ 0B            |             |
+-------------------+-------+----------------+-----------------+-------------+

*****************************************
 Create a Pipeline for Training Data
*****************************************

To streamline your training data processing, we'll set up a pipeline in Pachyderm. This pipeline will merge the data from the train and test repositories, then compress them into a tar file. This provides easy data access and also serves as a convenient checkpoint for data cleanup or transformations.

Execute the following commands to establish the pipeline:

.. code:: bash

   !pachctl create pipeline -f ./pachyderm/pipelines/compress/compress.json
   !pachctl list pipeline

Your expected pipeline list output should appear as:

TABLE

   PROJECT           NAME     VERSION INPUT                                                  CREATED       STATE / LAST JOB DESCRIPTION                                                          
   batch-inference-1 compress 1       (batch-inference-1/train:/ ⨯ batch-inference-1/test:/) 2 seconds ago running / -      A pipeline that compresses images from the train and test data sets. 

To verify the repositories:

.. code:: bash

   !pachctl list repo

The anticipated repo list should display:

+-------------------+----------+------------------+-----------------+------------------------------------------------------+
| PROJECT           | NAME     | CREATED          | SIZE (MASTER)   | DESCRIPTION                                          |
+===================+==========+==================+=================+======================================================+
| batch-inference-1 | compress | 5 seconds ago    | ≤ 0B            | Output repo for pipeline batch-inference-1/compress. |
+-------------------+----------+------------------+-----------------+------------------------------------------------------+
| batch-inference-1 | train    | 2 minutes ago    | ≤ 0B            |                                                      |
+-------------------+----------+------------------+-----------------+------------------------------------------------------+
| batch-inference-1 | test     | 2 minutes ago    | ≤ 0B            |                                                      |
+-------------------+----------+------------------+-----------------+------------------------------------------------------+


*****************************************
 Create a Pipeline for Training Data
*****************************************

To streamline your training data processing, we'll set up a pipeline in Pachyderm. This pipeline will merge the data from the train and test repositories, then compress them into a tar file. This provides easy data access and also serves as a convenient checkpoint for data cleanup or transformations.

Execute the following commands to establish the pipeline:

.. code:: bash

   !pachctl create pipeline -f ./pachyderm/pipelines/compress/compress.json
   !pachctl list pipeline

Your expected pipeline list output should appear as:

TABLE

   PROJECT           NAME     VERSION INPUT                                                  CREATED       STATE / LAST JOB DESCRIPTION                                                          
   batch-inference-1 compress 1       (batch-inference-1/train:/ ⨯ batch-inference-1/test:/) 2 seconds ago running / -      A pipeline that compresses images from the train and test data sets. 

To verify the repositories:

.. code:: bash

   !pachctl list repo

The anticipated repo list should display:

+-------------------+----------+------------------+-----------------+------------------------------------------------------+
| PROJECT           | NAME     | CREATED          | SIZE (MASTER)   | DESCRIPTION                                          |
+===================+==========+==================+=================+======================================================+
| batch-inference-1 | compress | 5 seconds ago    | ≤ 0B            | Output repo for pipeline batch-inference-1/compress. |
+-------------------+----------+------------------+-----------------+------------------------------------------------------+
| batch-inference-1 | train    | 2 minutes ago    | ≤ 0B            |                                                      |
+-------------------+----------+------------------+-----------------+------------------------------------------------------+
| batch-inference-1 | test     | 2 minutes ago    | ≤ 0B            |                                                      |
+-------------------+----------+------------------+-----------------+------------------------------------------------------+

*******************************************************
 Use the Compress Repo Data to Train Your Models
*******************************************************

By leveraging the power of a Determined cluster, you can efficiently train your models based on the data stored and versioned in Pachyderm. For this, it's vital to inform Determined about the location of your Pachyderm data. This includes providing details about the Pachyderm host, port, project, repo, and branch. 

When you create the experiment, remember to modify the Pachyderm host and port to align with the actual host and port of your Pachyderm cluster.

To view the configuration for the experiment, run:

.. code:: bash

   !cat ./determined/train.yaml

The configuration should resemble:

.. code-block:: yaml

   description: catdog_single_train
   data:
     pachyderm:
       host: PACHD_HOST
       port: PACHD_PORT
       project: batch-inference-1
       repo: compress
       branch: master
   hyperparameters:
     learning_rate: 0.005
     global_batch_size: 16
     weight_decay: 1.0e-4
     nesterov: true
   searcher:
     name: single
     metric: accuracy
     max_length:
       batches: 100
     smaller_is_better: false
   entrypoint: model_def:CatDogModel
   scheduling_unit: 10
   min_validation_period:
     batches: 10

To create the experiment, run:

.. code:: bash

   !det experiment create ./determined/train.yaml ./determined --config data.pachyderm.host=MacBook-Pro-3.local --config data.pachyderm.port=80

Upon successful creation, you should expect the following output:

OUTPUT

   Preparing files to send to master... 19.0KB and 11 files
   Created experiment 10

******************************************************
 Download Checkpoints from Determined
******************************************************

After training your model using Determined, you'll likely want to access and retain the best-performing checkpoints. By following the steps below, you can download the desired checkpoint and subsequently store it within a Pachyderm repo for future reference.

First, ensure you replace the trial id with the id of the recently completed trial:

.. code:: bash

   !det trial download 10 --best -o ./data/checkpoints/catdog1000

Upon execution, you should expect to see the following output:

.. code:: 

   Local checkpoint path: data/checkpoints/catdog1000

Now, let's create a new Pachyderm repository to store our models:

.. code:: bash

   !pachctl create repo models

You can verify the repository's creation by listing all available repos:

.. code:: bash

   !pachctl list repo

The table output should resemble:

+-------------------+----------+------------------+-----------------+------------------------------------------------------+
| PROJECT           | NAME     | CREATED          | SIZE (MASTER)   | DESCRIPTION                                          |
+===================+==========+==================+=================+======================================================+
| batch-inference-1 | models   | 2 seconds ago    | ≤ 0B            |                                                      |
+-------------------+----------+------------------+-----------------+------------------------------------------------------+
| batch-inference-1 | compress | 38 minutes ago   | ≤ 21.13MiB      | Output repo for pipeline batch-inference-1/compress. |
+-------------------+----------+------------------+-----------------+------------------------------------------------------+
| batch-inference-1 | train    | 40 minutes ago   | ≤ 17.36MiB      |                                                      |
+-------------------+----------+------------------+-----------------+------------------------------------------------------+
| batch-inference-1 | test     | 41 minutes ago   | ≤ 4.207MiB      |                                                      |
+-------------------+----------+------------------+-----------------+------------------------------------------------------+

Lastly, to add the checkpoint to your newly created repo, execute:

.. code:: bash

   !pachctl put file -r models@master:/catdog1000 -f ./data/checkpoints/catdog1000

***************************************************
 Create a Repo and Pipeline for Inferencing
***************************************************

Now that we have our trained model stored in the `models` repo, let's establish a new repository and pipeline dedicated to inferencing. This step allows for the model's utilization in predicting batches of files. Additionally, to enhance the processing speed and manage higher loads, we can introduce a parallelism specification in our pipeline spec.

Start by creating the `predict` repository:

.. code:: bash

   !pachctl create repo predict

To verify the creation, list all the available repositories:

.. code:: bash

   !pachctl list repo

The table output should be as follows:

+-------------------+----------+------------------+-----------------+-----------------------------------------------------+
| PROJECT           | NAME     | CREATED          | SIZE (MASTER)   | DESCRIPTION                                         |
+===================+==========+==================+=================+=====================================================+
| batch-inference-1 | predict  | 2 seconds ago    | ≤ 0B            |                                                     |
+-------------------+----------+------------------+-----------------+-----------------------------------------------------+
| batch-inference-1 | models   | 36 seconds ago   | ≤ 179.8MiB      |                                                     |
+-------------------+----------+------------------+-----------------+-----------------------------------------------------+
| batch-inference-1 | compress | 38 minutes ago   | ≤ 21.13MiB      | Output repo for pipeline batch-inference-1/compress.|
+-------------------+----------+------------------+-----------------+-----------------------------------------------------+
| batch-inference-1 | train    | 41 minutes ago   | ≤ 17.36MiB      |                                                     |
+-------------------+----------+------------------+-----------------+-----------------------------------------------------+
| batch-inference-1 | test     | 41 minutes ago   | ≤ 4.207MiB      |                                                     |
+-------------------+----------+------------------+-----------------+-----------------------------------------------------+

Next, create the pipeline for prediction:

.. code:: bash

   !pachctl create pipeline -f ./pachyderm/pipelines/predict/predict.json

Now, to confirm the pipeline's creation, you can list them:

.. code:: bash

   !pachctl list pipeline

The ensuing table output should resemble:

+-------------------+---------------------+---------+----------------------------------------------------------------------+----------------+------------------+----------------------------------------------------------------------------------------+
| PROJECT           | NAME                | VERSION | INPUT                                                                | CREATED        | STATE / LAST JOB | DESCRIPTION                                                                            |
+===================+=====================+=========+======================================================================+================+==================+========================================================================================+
| batch-inference-1 | predict-catdog      | 1       | (batch-inference-1/predict:/* ⨯ batch-inference-1/models:/*)         | 50 seconds ago | running /success | A pipeline that classifies                                                             |
+-------------------+---------------------+---------+----------------------------------------------------------------------+----------------+------------------+----------------------------------------------------------------------------------------+


*********************************************************
 Add Some Files for Pachyderm/Determined to Inference
*********************************************************

After setting up the pipeline, we can now push some files for the prediction. This is flexible; you can add any number of files to the `predict` repository at any time. Keep in mind that our pipelines will not only generate an image as output but also store the prediction result as a row in a CSV.

To add files for prediction, run:

.. code:: bash

   !pachctl put file -r predict@master -f ./data/predict/batch_10

********************************
 Add a Results Pipeline
********************************

Next, we'll set up a `results` pipeline. Its role is to gather all the predictions and then process them to generate various visualizations like charts. Additionally, it can store these predictions in a structured database format.

Start by creating the `results` pipeline:

.. code:: bash

   !pachctl create pipeline -f ./pachyderm/pipelines/results/results.json

To confirm the pipeline's creation and get an overview of all pipelines, list them:

.. code:: bash

   !pachctl list pipeline

+---------------------+---------------------+---------+-------------------------------------------------------+----------------+---------------------+-------------------------------------------------------------------------------------------------------------------+
| PROJECT             | NAME                | VERSION | INPUT                                                 | CREATED        | STATE / LAST JOB    | DESCRIPTION                                                                                                       |
+=====================+=====================+=========+=======================================================+================+=====================+===================================================================================================================+
| batch-inference-1   | results             | 1       | batch-inference-1/predict-catdog:/                    | 3 seconds ago  | running / -         | A pipeline that merges results from the predict pipelines.                                                        |
+---------------------+---------------------+---------+-------------------------------------------------------+----------------+---------------------+-------------------------------------------------------------------------------------------------------------------+
| batch-inference-1   | predict-catdog      | 1       | SOMETHING ELSE GOES HERE                              | 3 minutes ago  | running / success   | A pipeline that classifies images from the predict repo using models in the models repo.                          |
+---------------------+---------------------+---------+-------------------------------------------------------+----------------+---------------------+-------------------------------------------------------------------------------------------------------------------+
| batch-inference-1   | compress            | 1       | (batch-inference-1/train:/⨯ batch-inference-1/test:/) | 42 minutes ago | running / success   | A pipeline that compresses images from the train and test data sets.                                              |
+---------------------+---------------------+---------+-------------------------------------------------------+----------------+---------------------+-------------------------------------------------------------------------------------------------------------------+

****************************************************
Add More Files for Prediction and Results Pipelines
****************************************************

To watch all of the prediction and results pipelines run, add more files.

To do this, run the following commands:

.. code:: bash

   !pachctl put file -r predict@master -f ./data/predict/batch_5_2



************
 Next Steps
************

Congratulations! You've successfully set up Determined and Pachyderm on your machine. As you become more familiar with the tools, consider exploring advanced features and diving deeper into their documentation.

