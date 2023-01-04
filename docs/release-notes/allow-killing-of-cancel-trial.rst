:orphan:

**Improvements**

-  Trials: Trials can now be killed when in the ``STOPPING_CANCELED`` state. Previously if a trial
   did not implement preemption correctly and was canceled the trial did not stop and was unkillable
   until the preemption timeout of an hour.
