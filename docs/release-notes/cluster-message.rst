:orphan:

**New Features**

-  WebUI: Add the ability for administrators to use the CLI to set a message to be displayed on all
   pages of the WebUI (for example, ``det master cluster-message set -m "Your message"``). Optional
   flags are available for scheduling the message with a start time and an end time. Administrators
   can clear the message anytime using ``det master cluster-message clear``. Only one message can be
   active at a time, so setting a new message will replace the previous one.
