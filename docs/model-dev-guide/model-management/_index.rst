##################
 Model Management
##################

*****************
 Use Checkpoints
*****************

When a model is trained with Determined, checkpoints are automatically saved to external storage.
These checkpoints can then be exported for use outside Determined. See :ref:`use-trained-models` for
details.

*********************
 Archive Experiments
*********************

After training, you can archive experiments to clean up your list of experiments. Archiving is
designed to make it easier to organize experiments by omitting information about experiment runs
that are no longer relevant (e.g., training jobs that failed with an error or jobs submitted as part
of the model development process). When an experiment is archived, it is hidden from the default
view in both the WebUI and :ref:`the Determined CLI <cli-ug>`, but all of the metadata associated
with the experiment (including checkpoints) is preserved. An experiment can subsequently be
unarchived if desired, without losing any of the experiment's metadata.

********************
 Delete Checkpoints
********************

The best way to delete a checkpoint is to modify the garbage collection policy of the experiment
that created the checkpoint. For example, to delete *all* of the experiments associated with an
experiment, run:

.. code::

   det experiment set gc-policy --save-experiment-best 0 --save-trial-best 0 --save-trial-latest 0 <experiment-id>

***********************
 Manage Trained Models
***********************

Determined includes a built-in :doc:`model registry
</model-dev-guide/model-management/model-registry-org>` to manage trained models and their
respective versions.

.. toctree::
   :maxdepth: 1
   :hidden:
   :glob:

   ./*
