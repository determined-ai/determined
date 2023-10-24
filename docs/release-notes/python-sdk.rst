:orphan:

**New Features**

-  Python SDK: various new features and enhancements. A few highlights are listed below.
      -  Add support for downloading a zipped archive of experiment code
         (:meth:`Experiment.download_code
         <determined.experimental.client.Experiment.download_code>`).

      -  Add support for :class:`~determined.experimental.client.Project` and
         :class:`~determined.experimental.client.Workspace` as SDK objects.

      -  Surface more attributes to resource classes, including ``hparams`` and ``summary_metrics``
         for :class:`~determined.experimental.client.Trial`.

      -  Add support for fetching and filtering multiple experiments with
         :meth:`client.list_experiments <determined.experimental.client.list_experiments>`.

      -  Add support for filtering trial logs by timestamp and a query string using
         :meth:`Trial.iter_logs <determined.experimental.client.Trial.iter_logs>`.

      -  All resource objects now have a ``.reload()`` method that refreshes the resource's
         attributes from the server. Previously, attributes were most easily refreshed by creating
         an entirely new object.

-  Python SDK: all ``GET`` API calls now retry the request (maximum of 5 times) on failure.

**Deprecated Features**

-  Python SDK: several methods have been renamed for better API standardization.
      -  ``list_*`` and ``iter_*`` for methods returning a ``List`` and ``Iterator``, respectively.

      -  :class:`~determined.experimental.client.TrialReference` and
         :class:`~determined.experimental.client.ExperimentReference` are now
         :class:`~determined.experimental.client.Trial` and
         :class:`~determined.experimental.client.Experiment`.

-  Python SDK: consolidate various ways of fetching checkpoints.
      -  :meth:`Experiment.top_checkpoint
         <determined.experimental.client.Experiment.top_checkpoint>`, and
         :meth:`Experiment.top_n_checkpoints
         <determined.experimental.client.Experiment.top_n_checkpoints>` deprecated in favor of
         :meth:`Experiment.list_checkpoints
         <determined.experimental.client.Experiment.list_checkpoints>`.

      -  :meth:`Trial.get_checkpoints <determined.experimental.client.Trial.get_checkpoints>`,
         :meth:`Trial.top_checkpoint <determined.experimental.client.Trial.top_checkpoint>`, and
         :meth:`Trial.select_checkpoint <determined.experimental.client.Trial.select_checkpoint>`
         deprecated in favor of :meth:`Trial.list_checkpoints
         <determined.experimental.client.Trial.list_checkpoints>`.

-  Python SDK: deprecate resource ordering enum classes (``CheckpointOrderBy``,
   ``ExperimentOrderBy``, ``TrialOrderBy``, ``ModelOrderBy``) in favor of a singular shared
   :class:`~determined.experimental.client.OrderBy`.
