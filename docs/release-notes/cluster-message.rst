:orphan:

**New Features**

-  WebUI Cluster Message: Add a feature where administrators can set a message to be displayed on all pages
   of the WebUI. This message is set in the CLI like ``det master cluster-message set -m "Your message"``
   with optional flags for start time (for scheduling messages to display in the future) and end time
   (for clearing messages automatically once the end has been reached). The message can be cleared
   at any time with ``det master cluster-message clear``. Only one message can be active at a time,
   so setting a new message clears any previous message.
