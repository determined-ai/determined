:orphan:

**API Changes**

- Checkpoints: The checkpoint download API will now allow `application/x-tar` as a valid accept type in the request, and responds with an uncompressed tar file that should include a content-length in the headers.