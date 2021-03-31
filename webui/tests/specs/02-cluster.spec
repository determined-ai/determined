# Cluster
Tags: parallelizable

Specification to test the responsive design elements.

## Sign in

* Sign in as "user-w-pw" with "special-pw"
* Navigate to dashboard page

## Check Cluster elements

* Navigate to React page at "/cluster"

* Should show "1" resource pool cards

* Should show "1" agents in stats

## Check Cluster Historical Usage elements

* Navigate to React page at "/cluster/historical-usage"

* Page should contain "GPU Hours Allocated"
* Page should contain "GPU Hours by User"
* Page should contain "GPU Hours by Label"
* Page should contain "GPU Hours by Resource Pool"
* Page should contain "GPU Hours by Agent Label"

## Sign out

* Sign out
