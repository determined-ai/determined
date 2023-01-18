:orphan:

**Bug Fixes**

-  Agent: Fix a bug where if the flag ``agent_reattach_enabled`` was enabled and master was down
   while an active task's docker container failed, the task could get stuck in an unkillable running
   state.
