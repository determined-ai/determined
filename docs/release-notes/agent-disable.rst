:orphan:

**New Features**

-  Kubernetes: Kubernetes now supports agent enable and disable to prevent Determined from
   scheduling jobs on disabled nodes.

   Upgrading from a version before this feature to a version after this feature only on Kubernetes
   will cause queued allocations to be killed on upgrade. Users can pause queued experiments to
   avoid this.
