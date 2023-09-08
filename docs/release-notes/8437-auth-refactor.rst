:orphan:

**Removed Features**

-  **Breaking Change** Removed the accidentally-exposed Session object from the
   ``det.experimental.client`` namespace. It was never meant to be a public API and it was not
   documented in :ref:`python-sdk`, but was nonetheless exposed in that namespace. It was also
   available as a deprecated legacy alias, ``det.experimental.Session``. It is expected that most
   users use the Python SDK normally and are unaffected by this change, since the
   ``det.experimental.client``'s ``login()`` and ``Determined()`` are unaffected.

-  **Breaking Change** Add a new requirement for runtime configurations that there be a writable
   ``$HOME`` directory in every container. Previously, there was limited support for containers
   without a writable ``$HOME``, merely by coincidence. This change could impact users in scenarios
   where jobs were configured to run as the ``nobody`` user inside a container, instead of the
   ``det-nobody`` alternative recommended in :ref:`run-unprivileged-tasks`. Users combining non-root
   tasks with custom images not based on Determined's official images may also be affected. Overall,
   it is expected that few or no users are affected by this change.
