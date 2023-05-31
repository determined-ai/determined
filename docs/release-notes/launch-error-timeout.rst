:orphan:

**New Features**

-  AWS and GCP: Add `launch_error_timeout` and `launch_error_retries` provider configuration
   options.

   -  `launch_error_timeout`: Duration for which a provisioning error is valid. Tasks that are
      unschedulable in the existing cluster may be canceled. After the timeout period, the error
      state is reset. Defaults to `0s`.

   -  `launch_error_retries`: Number of retries to allow before registering a provider provisioning
      error with `launch_error_timeout`` duration. Defaults to `0`.
