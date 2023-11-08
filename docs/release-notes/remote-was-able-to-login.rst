:orphan:

**Bug Fixes**

-  Users: Fix an issue where if a user's remote status was edited through ``det user edit <username>
   --remote=true`` that user could still login through their username and password while they were
   expected to only be able to login through IDP integrations.
