# Non Scalar Metrics

Specification to test experiemnt page given non scalar metrics.

## Sign in

* Sign in as "user-w-pw" with "special-pw"

## Check that the page renders without errors

* Activate experiment "4"
* Await experiment "4" completion
* Navigate to experiment "4" page
* Require page to have "experiment 4, trial, summary"
* Require page to not have "error, fail, warn"

## Sign out

* Sign out
