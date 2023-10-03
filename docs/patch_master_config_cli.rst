:orphan:

**Breaking Change**

-  CLI: The old CLI command to patch master log config has been changed from ``det master config
   --log --level <log_level> --color <on/off>`` to ``det master set config log --level=<log_level>
   --color=<on/off>``. Earlier the acceptable log levels were "fatal", "error", "warn", "info",
   "debug" and "trace". Now the new acceptable log levels are "LOG_LEVEL_CRITICAL",
   "LOG_LEVEL_ERROR", "LOG_LEVEL_WARNING", "LOG_LEVEL_INFO", "LOG_LEVEL_DEBUG" and
   "LOG_LEVEL_TRACE".
