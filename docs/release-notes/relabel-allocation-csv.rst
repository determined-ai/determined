:orphan:

**Breaking Changes**

-  Tasks: The :ref:`historical usage <historical-cluster-usage-data>` CSV header row for slot-hours
   is now named ``slot_hours`` as it may also track allocation time for resource pools without GPUs.
   Also, this CSV now has an additional column providing the ``resource_pool`` for each allocation.
