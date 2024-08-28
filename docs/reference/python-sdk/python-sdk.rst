.. _python-sdk:

####################################
 Python SDK Client Module Reference
####################################

The client module exposes many of the same capabilities as the ``det`` CLI tool directly to Python
code with an object-oriented interface.

************
 ``Client``
************

.. automodule:: determined.experimental.client
   :members:
   :exclude-members: stream_trials_metrics, stream_trials_training_metrics, stream_trials_validation_metrics
   :member-order: bysource

*************
 ``OrderBy``
*************

.. autoclass:: determined.experimental.client.OrderBy
   :members:
   :member-order: bysource

.. _python-sdk-checkpoint:

****************
 ``Checkpoint``
****************

.. autoclass:: determined.experimental.client.Checkpoint
   :members:
   :member-order: bysource

****************
 ``Determined``
****************

.. autoclass:: determined.experimental.client.Determined
   :members:
   :exclude-members: stream_trials_metrics, stream_trials_training_metrics, stream_trials_validation_metrics
   :member-order: bysource

****************
 ``Experiment``
****************

.. autoclass:: determined.experimental.client.Experiment
   :members:
   :exclude-members: get_trials
   :member-order: bysource

******************
 ``DownloadMode``
******************

.. autoclass:: determined.experimental.client.DownloadMode
   :members:
   :member-order: bysource

***********
 ``Model``
***********

.. autoclass:: determined.experimental.client.Model
   :members:
   :exclude-members: get_metrics
   :member-order: bysource

*****************
 ``ModelSortBy``
*****************

.. autoclass:: determined.experimental.client.ModelSortBy
   :members:
   :member-order: bysource

******************
 ``ModelVersion``
******************

.. autoclass:: determined.experimental.model.ModelVersion
   :members:
   :member-order: bysource

*************
 ``Project``
*************

.. autoclass:: determined.experimental.client.Project
   :members:
   :member-order: bysource

******************
 ``ResourcePool``
******************

.. autoclass:: determined.experimental.client.ResourcePool
   :members:
   :member-order: bysource

***********
 ``Trial``
***********

.. autoclass:: determined.experimental.client.Trial
   :members:
   :exclude-members: stream_metrics, stream_training_metrics, stream_validation_metrics
   :member-order: bysource

******************
 ``TrialMetrics``
******************

.. autoclass:: determined.experimental.client.TrialMetrics

**********
 ``User``
**********

.. autoclass:: determined.experimental.client.User
   :members:
   :member-order: bysource

***************
 ``Workspace``
***************

.. autoclass:: determined.experimental.client.Workspace
   :members:
   :member-order: bysource
