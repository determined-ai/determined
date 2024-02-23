:orphan:

**Bug Fixes**

-  ResourceManager: Prevent connections from duplicate agents. Agent connection attempts will be
   rejected if there's already an active connection from a matching agent ID. This prevents and
   replaces previous behavior of stopping the running agent when a duplicate connection attempt is
   made (causing both connections to fail).
