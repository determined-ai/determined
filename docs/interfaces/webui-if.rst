.. _web-ui-if:

#######################
 Web Interface (WebUI)
#######################

The web interface, WebUI, provides a convenient way to create and monitor experiment progress.

(mention the CLI here)

********************
 Accessing the WebUI
********************

To access the Determined AI WebUI, follow these steps:

-  Open your preferred web browser.
-  Enter the default URL ``http://master-addr:8080`` in the address bar.
- Replace ``master-addr`` with the hostname or IP address of the master.
-  If TLS is enabled, use port ``8443`` instead of ``8080`` in the URL.
-  If a valid Determined session has not been established, WebUI automatically redirects to the
   login page.

************
 Logging In
************

To log in to the Determined AI WebUI, follow these steps:

-  Enter your username and password on the login page.
-  Click **Log In**.

If your credentials are valid, you will be redirected to your initial URL.

*************
 Signing Out
*************

To end the Determined session and sign out of the WebUI:

-  Click **Sign Out** at the top-right.

************
 Navigating
************

The Determined AI WebUI is organized into several pages, each with its own set of features. Here's
an overview of the main pages:

-  Home: From the home page, you can view your most recent experiments and launch JupyterLab.
-  JupyterLab: Lists all Jupyter notebooks and allows you to create and launch new notebooks.
-  Experiments: Experiments is a customizable page that lets you view all experiments. For example,
   you can apply batch operations, filter, sort and archive.
-  Model Registry: Description. All models. Ability to deploy models to production???
-  Tasks: Description.
-  Cluster: View, configure, and manage the compute resources available in your cluster.
-  Workspaces: All workspaces.

***************************
 Creating a New Experiment
***************************

To create a new experiment in the Determined AI WebUI, follow these steps:

-  Navigate to the "Experiments" page. Click the "New Experiment" button.
-  Enter a name for your experiment.
-  Choose a task type from the dropdown menu.
-  Choose a deep learning framework from the dropdown menu.
-  Choose a cluster configuration from the dropdown menu.
-  Configure any additional settings, such as hyperparameters and data.
-  Click the "Create" button.

********************************
 Monitoring Experiment Progress
********************************

The Determined AI WebUI provides real-time updates on experiment progress. Here's an overview of the
main features for monitoring experiment progress:

-  Experiment Details: This page provides detailed information about a specific experiment,
   including its status, metrics, and logs.
-  TensorBoard Integration: The Determined AI WebUI integrates with TensorBoard, allowing you to
   view real-time graphs of metrics and visualizations.
-  Training Plots: The Determined AI WebUI provides real-time updates on training plots, allowing
   you to visualize training progress over time.
-  Job Manager: The Determined AI WebUI provides a job manager, allowing you to view and manage all
   tasks associated with a specific experiment.

*********************************
 Deploying a Model to Production
*********************************

To deploy a model to production in the Determined AI WebUI, follow these steps:

-  Navigate to the "Models" page.
-  Select the model you want to deploy.
-  Click the "Deploy" button.
-  Choose a deployment configuration, such as the number of replicas and the deployment environment.
-  Configure any additional settings, such as network policies and resource limits.
-  Click the "Deploy" button.

*************************
 Viewing Experiment Logs
*************************

The Determined AI WebUI provides detailed logs for each experiment. Here's an overview of how to
view experiment logs:

-  Navigate to the "Experiments" page.
-  Click on the experiment you want to view logs for.
-  Click the "Logs" tab.
-  Select the log type you want to view, such as "Task Logs" or "System Logs".

Use the search bar to search for specific keywords or phrases. Use the filter options to narrow down
the log output.

****************************
 Viewing Experiment Metrics
****************************

The Determined AI WebUI provides real-time updates on experiment metrics. Here's an overview of how
to view experiment metrics:

-  Navigate to the "Experiments" page.
-  Click on the experiment you want to view metrics for.
-  Click the "Metrics" tab.
-  Select the metric you want to view, such as "Training Loss" or "Validation Accuracy".
-  Use the time range selector to adjust the time window for the metrics.
-  Use the filter options to narrow down the metric output.
