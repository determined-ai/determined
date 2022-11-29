:orphan:

**Bug Fixes**

-  Python SDK: Model Registry call ``model.get_version(version)`` did not work when a specific
   version was passed. This is now fixed.

**Breaking Changes**

-  REST APIs: ``GetModelVersion``, ``PatchModelVersion``, ``DeleteModelVersion`` APIs now take
   sequential model version number ``model_version_num`` instead of a surrogate key
   ``model_version_id``.
