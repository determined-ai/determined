#####################################
 Monitor Experiment Through Webhooks
#####################################

This section includes user guides for how to set up notifications for experiments.

Determined now supports webhooks for sending notification to monitor experiment state change.

**********
 Security
**********

Determined utilizes ``signed_payload`` and ``timestamp`` headers. You will need to configure a
``webhook_signing_key`` in master configuration.

.. code::

   security:
       webhook_signing_key: <signing_key>

.. note::

   ``webhook_signing_key`` will be automatically generated if not provided, you can access it
   through ``api/v1/master/config``

For every webhook request Determined will generate two headers ``X-Determined-AI-Signature``, and
``X-Determined-AI-Signature-Timestamp`` which you can inspect to verify each request to their
webhook endpoint.

-  The ``X-Determined-AI-Signature-Timestamp`` will represent the time at which the request was
   generated and sent. You can choose to inspect this timestamp and decide whether to discard any
   requests with a timestamp that is too distant from the current time at their discretion.

-  The ``X-Determined-AI-Signature`` will be a representation of a "signed" request payload. The signed request payload will be generated in the following way:
      -  Combine the timestamp in ``X-Determined-AI-Signature-Timestamp``, the comma character ","
         and the request body, which will be the entire event payload.
      -  Create an HMAC using SHA256 hashing, with the ``webhook_signing_key``, and the event
         payload from previous step.

-  You can then check to make sure the ``X-Determined-AI-Signature`` header value and the generated
   signed payload match.

****************
 Create Webhook
****************

To create a webhook, navigate to ``/det/webhooks`` and click on the top right corner button "New
Webhook"

.. image:: /assets/images/webhook.png
   :width: 100%

.. note::

   You must have the relevant permission to be able to view this page, consult system admin if you
   are unsure about your permissions.

At the popped-up modal, input:

-  URL: webhook URL.
-  Type: choose between ``Default`` or ``Slack``. Type ``Slack`` can automatically format message
   content for better readability on Slack.
-  Trigger: choose which state change of experiment you want to monitor, and this field only
   supports ``Completed`` or ``Error`` for now.

.. image:: /assets/images/webhook_modal.png
   :width: 100%

Once created, the selected event of all available experiments will trigger the defined webhook URL.

**************
 Test Webhook
**************

To test a webhook, click on the triple dots on the right of webhook record to expand available
actions.

.. image:: /assets/images/webhook_action.png
   :width: 100%

Clicking on "Test Webhook" would trigger a test event to be sent to the defined webhook URL, with a
similar mock payload as stated below:

.. code::

   {
       "event_id":"1ac7d0b2-a4af-458b-a099-2326240088f6",
       "event_type":"experiment_completed",
       "timestamp":1662562300,
       "event_data":{
           "experiment": {
               "id":1,
               "state": "COMPLETED",
               "name": "cifar10_pytorch_const profiler",
               "duration": 18400,
               "resource_pool": "A100 Production",
               "slots":24,
               "workspace": {
                   "name": "Autonomous Vehicles",
                   "id": 1
               },
               "project": {
                   "name": "Light detection",
                   "id": 12
               }
           }
       }
   }

****************
 Delete Webhook
****************

To delete a webhook, click on the triple dots on the right of webhook record to expand available
actions.

.. note::

   Currently we do not support editing webhooks.

.. toctree::
   :caption: Notification
   :hidden:

   zapier
   slack
