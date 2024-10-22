.. _web-ui-if:

#######################
 Web Interface (WebUI)
#######################

.. meta::
   :description: Discover how to create and monitor experiment progress, track experiments, evaluate model performance, and organize your experiments into projects and workspaces using the Determined WebUI.

You can use the WebUI to create and monitor experiment progress, track experiments, evaluate model
performance, organize your experiments into projects and workspaces, start a Jupyter Notebook, and
more.

*****************
 Getting Started
*****************

To access the WebUI, go to the default URL ``http://master-addr:8080`` in your web browser, where
``master-addr`` is the hostname or IP address of the master. If Transport Layer Security (TLS) is
enabled, use port ``8443`` instead.

If you have not yet established a valid Determined session, the WebUI will automatically redirect
you to the sign-in page. Once you sign in, you will be redirected to the initial URL you entered.

To end the Determined session and sign out of the WebUI, select your profile name in the upper left
corner and choose **Sign Out**.

************************
 Creating an Experiment
************************

If you have an existing experiment or trial, you can use the WebUI to create an experiment. You'll
need to use the CLI to create a new experiment. To learn how to create a new experiment, visit
:ref:`experiments`.

After creating an experiment, you can use the WebUI to track its progress in real-time. The WebUI
allows you to monitor metrics, analyze results, and maintain full oversight of your experiments,
providing a clear and detailed view of the entire process.

**********************************************************
 Organizing Your Experiments into Projects and Workspaces
**********************************************************

The WebUI lets you organize your experiments into projects and workspaces. To learn more about
Projects and Workspaces, visit :ref:`workspaces`.

*******************************************
 Configuring the Behavior of an Experiment
*******************************************

Many experiment configuration settings can be performed via the WebUI. To learn more about
configuring an experiment via a YAML file, visit :ref:`experiment-config-reference`.

**************************************************
 Evaluating Model Performance and Experiment Runs
**************************************************

You can access training and validation performance information via the WebUI to evaluate your model.
To see model evaluation in action, follow the steps described in the :ref:`pytorch-mnist-tutorial`.

You can also compare single trials with various datasets, parameters, and settings so that you can
choose the best model. This simplified flat trials view lets you perform a quick metric evaluation
by selecting two or more runs to compare. This feature can be toggled on or off in :ref:`user
settings <web-ui-user-settings>`.

To start, select two or more runs and then select **Compare**:

.. image:: /assets/images/webui-runs.png
   :alt: WebUI showing list of runs for comparison

|

In the compare view, choose a tab for comparing metrics, hyperparameters, and other details of the
runs you selected.

.. image:: /assets/images/webui-runs-compared.png
   :alt: WebUI showing selected runs with hyperparameters compared

|

*************************************
 Adding Models to the Model Registry
*************************************

You can use the WebUI to create and edit models and add models to the model registry. You can also
use the WebUI to edit model metadata. To find out more, visit :ref:`organizing-models`.

***************************************
 Viewing Historical Cluster Usage Data
***************************************

You can use the WebUI to view :ref:`historical-cluster-usage-data`.

**************************
 Viewing Cluster Topology
**************************

To view a resource pool's node and GPU distribution, and find out how many GPUs are currently in
use, follow these steps:

#. Navigate to Resource Pools.

   From the left navigation pane, select **Cluster** to view **Resource Pools**.

#. Select a Resource Pool.

   In the resource pool details page, you will find a **Topology** section visible if agents or
   nodes are provisioned. If no agents or nodes are provisioned, the **Topology** section will not
   be visible.

   This view displays a visual representation of each node, including its unique identifier, and the
   number of available slots on each node.

#. View Active and Used Slots.

   In the topology visualization, all active or used slots will be highlighted in blue, making it
   easy to distinguish between available and occupied resources.

***********************************
 Managing User Accounts and Groups
***********************************

The ``admin`` user manages user authentication including creating and managing users. To learn more,
visit :ref:`users`.

With the Determined Enterprise Edition, you can also create and manage user groups. To learn more,
visit :ref:`rbac`.

.. _web-ui-user-settings:

************************
 Managing User Settings
************************

User settings allow you to manage profile settings, preferences, and shortcuts. You can also toggle
experimental features on or off and access advanced features.

To view user settings:

-  Select your profile name in the upper left corner and choose **Settings**.

To change user settings:

-  Select the edit icon.
-  Make changes to the setting.
-  Confirm the changes by selecting the checkmark.

To revert to default settings:

-  Navigate to the Advanced section of the user settings.
-  Select **I know what I'm doing**.
-  Select **Reset to Default**.
-  Confirm you want to reset all user settings to their default values.

****************************************
 Selecting a Table Density (Row Height)
****************************************

In the Preferences section of your user settings, you can set the table density so that the rows are
shorter or taller.

********************************************************
 Toggling Experimental (Pre-Release) Features On or Off
********************************************************

In the Experimental section of your user settings, you can turn experimental features on or off.
However, if you don't know what the feature is referring to or the possible impact, you likely
should not turn it on.

.. caution::

   Experimental features are pre-release features. They can be changed or removed at any time.

***********************
 Configuring Telemetry
***********************

To find what kind of anonymous information the WebUI collects, visit
:ref:`common-configuration-options`.

************************************
 Viewing and Managing the Job Queue
************************************

To find out how to view and modify the Job Queue in the WebUI, start with :ref:`job-queue`.

*****************************
 Starting a Jupyter Notebook
*****************************

You can start :ref:`notebooks` from the WebUI.

***********************
 Launching TensorBoard
***********************

You can launch TensorBoard from the WebUI. To learn how, visit :ref:`tensorboards`.

*****************************
 Displaying a Banner Message
*****************************

Administrators can create a banner message to alert users about important information, such as
maintenance, setting a password, or other announcements. This message will be displayed on the
header of every page in the WebUI for the configured duration. Commands include ``help``, ``clear``,
``get``, and ``set``.

**Prerequisites**

-  Install the :ref:`CLI <cli-ug>`.

**Prepare the Message**

Prepare the maintenance message using the CLI command, ``det master cluster-message set``.

-  For example, the following command creates a maintenance message with a start and end date (which
   must be expressed in UTC):

      .. code:: bash

         det master cluster-message set --message "Scheduled maintenance on Dec 1st from 10pm CST to 11pm CST." --start "2024-12-02-04:00:00Z" --end "2024-12-02-05:00:00Z"

-  You can also express the end date as a duration:

      .. code:: bash

         det master cluster-message set --message "Please change your password by Jan 1, 2025" --duration 14d

**Verify the Message**

Verify the message with the following command:

   .. code:: bash

      det master cluster-message get

**Clear the Message**

Clear the message with the following command:

   .. code:: bash

      det master cluster-message clear

********************************
 Viewing and Filtering Metadata
********************************

You can use the WebUI to view and filter experiment runs based on logged metadata. For a tutorial on
how to log metadata, visit :ref:`metadata-logging-tutorial`.

-  In the Overview tab of the experiment, you can filter and sort runs based on metadata values
   using the filter menu.
-  In the experiment's Runs view, metadata columns are displayed alongside other experiment
   information.
-  On the Run details page, you'll find the "Metadata" section under the "Overview" tab, displaying
   all logged metadata for that run.
-  To download the metadata in JSON format, click the "Download" button.

To filter runs based on metadata:

#. In the Runs view, click on the filter icon.
#. Select a metadata field from the dropdown menu.
#. Choose a condition (is, is not, or contains) and enter a value.

Note: Array-type metadata can be viewed but cannot be used for sorting or filtering.
