:orphan:

**Deprecated Features**

-  API: The ``SummarizeTrial`` endpoint is removed in favor of ``CompareTrials``; send a similar
   request with the `trial_id` parameter replaced by the `trial_ids` array.
-  API: The ``scale`` parameter is removed from ``CompareTrialsRequest``; this was used only for
   lttb downsampling which has since been replaced.
