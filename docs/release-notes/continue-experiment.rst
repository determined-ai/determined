:orphan:

**New Features**

-  Experiments: Add an experiment continue feature via a CLI command ``det e continue
   <experiment-id>``. This allows users to resume or recover training for an experiment whether it
   previously succeeded or failed. This is limited to single-searcher experiments and using it may
   prevent the user from replicating the continued experiment's results.
