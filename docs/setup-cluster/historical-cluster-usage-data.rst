.. _historical-cluster-usage-data:

###############################
 Historical Cluster Usage Data
###############################

Our goal is to give users insights on how their Determined cluster is used. Historical cluster usage
is measured in the number of compute hours allocated by Determined. Note that this is not based on
resource utilization, so if a user gets 1 GPU allocated but only utilizes 20% of the GPU, we would
still report one GPU hour.

.. warning::

   The total used compute hours reported by Determined may be less than the hours reported by the
   cloud because we do not include the time that the slots are idle (e.g., time waiting for a GPU to
   spin up, or when a GPU is not scheduled with any jobs) in that.

.. warning::

   Our data is aggregated by Determined metadata (e.g., label, user). This aggregation is performed
   nightly, so any data visualized on the WebUI or downloaded via the endpoint is fresh as of the
   last night. It will not reflect changes to the metadata of a previously run experiment (e.g.,
   labels) until the next nightly aggregation.

*********************
 WebUI Visualization
*********************

We build WebUI visualizations for a quick snapshot of the historical cluster usage:

.. image:: /assets/images/historical-cluster-usage-data.png
   :width: 100%
   :alt: WebUI showing historical cluster usage data

************************
 Command-line Interface
************************

Alternatively, you can use the :ref:`CLI <cli-ug>` or the API endpoints to download the resource
allocation data for analysis:

-  ``det resources raw <start time> <end time>``: get raw allocation information, where the times
   are full times in the format yyyy-mm-ddThh:mm:ssZ.
-  ``det resources aggregated <start date> <end date>``: get aggregated allocation information,
   where the dates are in the format yyyy-mm-dd.
