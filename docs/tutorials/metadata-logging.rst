.. _metadata-logging-tutorial:

############################
 Arbitrary Metadata Logging
############################

Arbitrary Metadata Logging enhances your experiment tracking capabilities by allowing you to:

#. Log custom metadata specific to your experiments.
#. View logged metadata in the WebUI for each trial.
#. Filter and sort experiment runs based on custom metadata.
#. Compare and analyze experiments using custom metadata fields.

By leveraging this feature, you can capture and analyze experiment-specific information beyond
standard metrics, leading to more insightful comparisons and better experiment management within the
Determined platform.

******************
 Example Use Case
******************

This example creates an arbitrary metadata field ``effectiveness`` and then (does something else).

**Section Title**

#. Run an experiment to create a :ref:`single-trial run <qs-webui-concepts>`.
#. Note the Run ID, e.g., Run 110.
#. Navigate to the cluster address for your training environment, e.g., **http://localhost:8080/**.
#. In the WebUI, click **API(Beta)** in the left navigation pane.
#. Execute the following PostRunMetadata ``/Internal/PostRunMetadata``

.. code:: bash

   {
       "runId": 110,
       "metadata": {
        "effectiveness": 20
   }
   }

Next, we'll filter our runs by a specific metadata condition.

**Filter by Metadata**

#. In the WebUI, select your experiment to view the Runs table.
#. Select the **Filter**.
#. In **Show runs...**, select your metadata field from the dropdown menu.
#. Choose a condition (e.g., is, is not, or contains) and enter a value.

.. image:: /assets/images/webui-runs-metadata-filter.png
   :alt: Determined AI metadata filter for runs for an experiment

Finally, let's view the logged metadata for our run.

**View Metadata**

To view the logged metadata:

#. In the WebUI, navigate to your experiment.
#. Click on the run you want to inspect.
#. In the Run details page, find the "Metadata" section under the "Overview" tab.

****************************
 Performance Considerations
****************************

When using Arbitrary Metadata Logging, consider the following:

-  Metadata is stored efficiently for fast retrieval and filtering.
-  Avoid logging very large metadata objects, as this may impact performance.
-  Use consistent naming conventions for keys to make filtering and sorting easier.
-  For deeply nested JSON structures, filtering and sorting are supported at the top level.

************
 Next Steps
************

-  Experiment with logging different types of metadata in your trials.
-  Use the filtering and sorting capabilities in the WebUI to analyze your experiments.
-  Integrate metadata logging into your existing Determined AI workflows to enhance your experiment
   tracking.

For more tutorials and guides, visit the :ref:`tutorials-index`.
