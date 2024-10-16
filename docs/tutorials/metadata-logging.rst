.. _metadata-logging-tutorial:

############################
 Arbitrary Metadata Logging
############################

This tutorial demonstrates how to use Arbitrary Metadata Logging in Determined AI to log custom metadata for your experiments.

**Why Use Arbitrary Metadata Logging?**

Arbitrary Metadata Logging allows you to:

- Capture experiment-specific information beyond standard metrics
- Compare and analyze custom data across experiments
- Filter and sort experiments based on custom metadata

******************
 Logging Metadata
******************

You can log metadata using the Determined Core API. Here's how to do it in your training code:

1. Import the necessary module:

   .. code:: python

      from determined.core import Context

2. In your trial class, add a method to log metadata:

   .. code:: python

      def log_metadata(self, context: Context):
          context.train.report_metadata({
              "dataset_version": "MNIST-v1.0",
              "preprocessing": "normalization",
              "hardware": {
                  "gpu": "NVIDIA A100",
                  "cpu": "Intel Xeon"
              }
          })

3. Call this method in your training loop:

   .. code:: python

      def train_batch(self, batch: TorchData, epoch_idx: int, batch_idx: int):
          # Existing training code...
          
          if batch_idx == 0:
              self.log_metadata(self.context)

          # Rest of the training code...

This example logs metadata at the beginning of each epoch. Adjust the frequency based on your needs.

*******************************
 Viewing Metadata in the WebUI
*******************************

To view logged metadata:

1. Open the WebUI and navigate to your experiment.
2. Click on the trial you want to inspect.
3. In the trial details page, find the "Metadata" section under the "Overview" tab.

***********************************
 Filtering and Sorting by Metadata
***********************************

The :ref:`Web UI <web-ui-if>` allows you to filter and sort experiments based on logged metadata:

1. Navigate to the Experiments List page in the WebUI.
2. Click on the filter icon.
3. Select a metadata field from the dropdown menu.
4. Choose a condition (is, is not, or contains) and enter a value.
5. Click "Apply" to filter the experiments based on the metadata.

For more detailed instructions on filtering and sorting, refer to the WebUI guide:

Performance Considerations
==========================

When using Arbitrary Metadata Logging, consider the following:

- Metadata is stored efficiently for fast retrieval and filtering.
- Avoid logging very large metadata objects, as this may impact performance.
- Use consistent naming conventions for keys to make filtering and sorting easier.
- For deeply nested JSON structures, filtering and sorting are supported at the top level.

Example Use Case
================

Let's say you're running experiments to benchmark different hardware setups. For each run, you might log:

.. code:: python

   def log_hardware_metadata(self, context: Context):
       context.train.report_metadata({
           "hardware": {
               "gpu": "NVIDIA A100",
               "cpu": "Intel Xeon",
               "ram": "64GB"
           },
           "software": {
               "cuda_version": "11.2",
               "python_version": "3.8.10"
           },
           "runtime_seconds": 3600
       })

You can then use these logged metadata fields to:

1. Filter for experiments that ran on a specific GPU model.
2. Compare runtimes across different hardware configurations.
3. Analyze the impact of software versions on performance.

Summary
=======

Arbitrary Metadata Logging enhances your experiment tracking capabilities by allowing you to:

1. Log custom metadata specific to your experiments.
2. View logged metadata in the WebUI for each trial.
3. Filter and sort experiments based on custom metadata.
4. Compare and analyze experiments using custom metadata fields.

By leveraging this feature, you can capture and analyze experiment-specific information beyond standard metrics, leading to more insightful comparisons and better experiment management within the Determined AI platform.

Next Steps
==========

- Experiment with logging different types of metadata in your trials.
- Use the filtering and sorting capabilities in the WebUI to analyze your experiments.
- Integrate metadata logging into your existing Determined AI workflows to enhance your experiment tracking.

For more tutorials and guides, visit the :ref:`tutorials-index`.
