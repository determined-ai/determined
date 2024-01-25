:orphan:

**Deprecated Features**

-  API: The experiment API object in a future version will have its ``config`` field removed to
   improve performance of the system. A new ``config`` field is added now to the response of
   ``/api/v1/experiments/{experiment_id}`` that can be used as a replacement.

   If you are not calling the APIs manually there will be no impact to you.
