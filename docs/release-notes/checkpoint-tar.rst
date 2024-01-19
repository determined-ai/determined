:orphan:

**API Changes**

-  Checkpoints: The checkpoint download endpoint will now allow the use of `application/x-tar`` as
   an accepted content type in the request. It will provide a response in the form of an
   uncompressed tar file, complete with content-length information included in the headers.
