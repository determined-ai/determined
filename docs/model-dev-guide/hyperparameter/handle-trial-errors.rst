#################################################
 Handle Trial Errors and Early Stopping Requests
#################################################

When a trial encounters an error or fails unexpectedly, Determined will restart it from the latest
checkpoint up to some maximum number of times, which is configured by :ref:`max_restarts
<max-restarts>` in the experiment configuration. After Determined reaches ``max_restarts``, any
further trials that fail will be marked as errored and will not be restarted. For the :ref:`adaptive
(ASHA) <topic-guides_hp-tuning-det_adaptive-asha>` search method, which adapts to validation metric
values, we do not continue training errored trials, even if the search method would typically call
for us to continue training. This behavior is useful when some parts of the hyperparameter space
result in models that cannot be trained successfully (e.g., the search explores a range of batch
sizes and some of those batch sizes cause GPU OOM errors). An experiment can complete successfully
as long as at least one of the trials within it completes successfully.

Trial code can also request that training be stopped early, e.g., via a framework callback such as
`tf.keras.callbacks.EarlyStopping
<https://www.tensorflow.org/api_docs/python/tf/keras/callbacks/EarlyStopping>`__ or manually by
calling :meth:`determined.TrialContext.set_stop_requested`. When early stopping is requested,
Determined will finish the current training or validation workload and checkpoint the trial. Trials
that are stopped early are considered to be "completed", whereas trials that fail are marked as
"errored".
