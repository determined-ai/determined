:orphan:

**Improvements**

-  **Breaking Change:** ``det deploy aws`` by default now configures agent instances to
   automatically shut down if they lose their connection to the master. The
   ``--no-shut-down-agents-on-connection-loss`` option can be used to turn off this behavior.
