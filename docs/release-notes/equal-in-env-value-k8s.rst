:orphan:

**Bug Fixes**

-  Kubernetes: Fix an issue where environment variables with an equal character in the value such as
   ``func=f(x)=x`` were processed incorrectly in Kubernetes.

**Improvements**

-  Kubernetes: Empty environment variables can now be specified in Kubernetes while before they
   would throw an error.
