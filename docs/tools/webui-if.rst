.. _web-ui-if:

#######################
 Web Interface (WebUI)
#######################

.. meta::
   :description: Discover how to create and monitor experiment progress, organize your experiments into projects and workspaces, start a Jupyter Notebook, and more using the Determined WebUI.

You can use the WebUI to create and monitor experiment progress, organize your experiments into
projects and workspaces, start a Jupyter Notebook, and more.

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

***************************************
 Evaluating the Performance of a Model
***************************************

You can access training and validation performance information via the WebUI. To see model
evaluation in action, follow the steps described in the :ref:`pytorch-mnist-tutorial`.

*************************************
 Adding Models to the Model Registry
*************************************

You can use the WebUI to create and edit models and add models to the model registry. You can also
use the WebUI to edit model metadata. To find out more, visit :ref:`organizing-models`.

***************************************
 Viewing Historical Cluster Usage Data
***************************************

You can use the WebUI to view :ref:`historical-cluster-usage-data`.

***********************************
 Managing User Accounts and Groups
***********************************

The ``admin`` user manages user authentication including creating and managing users. To learn more,
visit :ref:`users`.

With the Determined Enterprise Edition, you can also create and manage user groups. To learn more,
visit :ref:`rbac`.

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
