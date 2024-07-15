:orphan:

**Breaking Changes**

-  Tasks: The :ref:`historical usage <historical-cluster-usage-data>` CSV file has been updated. The
   header row for slot-hours is now named ``slot_hours`` instead of ``gpu_hours`` to accurately
   reflect the allocation time for resource pools including those without GPUs. In addition, a new
   column, ``resource_pool``, has been added to provide the resource pool for each allocation.
