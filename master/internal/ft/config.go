package ft

/*

log_pattern_actions:
  - pattern: 'Cuda OOM'
    policy:
      type: on_failure_dont_retry
  - pattern: 'ECC error':
    policy:
      type: on_failure_exclude_node:
      retries: 5 # This might be overkill here we could just pick some number like 5
/*

- define the config struct to be added to task container defaults
	- pattern(s) to action mapping
	- define actions (or translate from proto definition)
- validate config
- merge user and system defined configs
	- provide option for users to see or override system defined ones?
*/
