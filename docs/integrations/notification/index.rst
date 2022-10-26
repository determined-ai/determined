########################################
 Monitoring Experiment Through Webhooks
########################################

Monitoring experiment status is a vital part of working with Determined. In order to integrate
Determined into your existing workflows, you can make use of webhooks to update other systems,
receive emails, slack messages, and more when an experiment is updated.

.. _webhook_security:

*****************************
 Security and Signed Payload
*****************************

Each Webhook request will include a signed payload that users can use to verify that webhook calls
from Determined are genuine. This will require that users define a ``webhook_signing_key`` for
signing. If a webhook signing key is not provided then one will be generated for the user.

Setting the Key
===============

The signing key can be set in the following ways:

#. In the master yaml configuration file by adding:

.. code::

   webhooks:
      signing_key: <signing_key>

2. By setting a ``DET_WEBHOOKS_SIGNING_KEY`` environment variable.
#. By specifying a ``---security-webhooks-signing-key`` flag.

Retrieving the Key
==================

The key can be found in the cluster configuration. For example it will be returned in
``api/v1/master/config``.

Event Payload
=============

Currently we will support two separate types of Webhooks ``Slack`` and ``Default``. A payload for a
``Default`` webhook will contain information about the event itself, the trigger for the event, and
the appropriate entity that triggered the event. The shape of ``event_data`` is determined by
``event_type``, that this is an example payload for ``EXPERIMENT_STATE_CHANGE``, and that other
types may be structured differently.

.. code::

   {
      "event_id": "4cd26e62-60c6-4a8b-8d03-7629091a4ef5".   // unique event uuid,
      "event_type": "EXPERIMENT_STATE_CHANGE" // the trigger type for the event,
      "timestamp": 1665689991 // the timestamp of the event occurence,
      "condition": {.
         "state": "COMPLETED". // the condition that triggered the event for now the condition will always be of this format and contain either "COMPLETED" or "ERRORED"
      },
      "event_data": {
         "experiment": {
            "id": 41,
            "state": "COMPLETED",
            "name": "cifar10_pytorch_const",
            "slots": 2, // slots per trial
            "duration": 41, // experiment duration measured in seconds
            "resource_pool": "default",
            "workspace": "test workspace", // experiment workspace name
            "project": "test project" // experiment project name
         }
      }
   }

Signed Payload
==============

For every webhook request Determined will generate two headers ``X-Determined-AI-Signature``, and
``X-Determined-AI-Signature-Timestamp`` which can be used to verify each request to a webhook
endpoint.

-  The ``X-Determined-AI-Signature-Timestamp`` will represent the time at which the request was
   generated and sent. You can choose to inspect this timestamp and decide whether to discard any
   requests with a timestamp that is too distant from the current time.

-  The ``X-Determined-AI-Signature`` will be a representation of a "signed" request payload. The signed request payload will be generated in the following way:
      -  Combine the timestamp in ``X-Determined-AI-Signature-Timestamp``, the comma character ","
         and the request body, which will be the entire event payload.
      -  Create an HMAC using SHA256 hashing, with the ``webhook_signing_key``, and the event
         payload from previous step.

-  You can then check to make sure the ``X-Determined-AI-Signature`` header value and the generated
   signed payload match.

Below is an example of handling signed payload in python

.. image:: /assets/images/webhook_security_eg.png
   :width: 100%

*******************
 Creating Webhooks
*******************

To create a webhook, navigate to ``/det/webhooks`` or click on the "Webhooks" item in navigation
side menu, and click on the top right corner button "New Webhook"

.. image:: /assets/images/webhook.png
   :width: 100%

.. note::

   You must have the relevant permission to be able to view this page, consult system admin if you
   are unsure about your permissions.

At the modal input:

-  URL: webhook URL.
-  Type: ``Default`` or ``Slack``. The ``Slack`` type can automatically format message content for
   better readability on Slack.
-  Trigger: the experiment state change you want to monitor, either ``Completed`` or ``Error``.

.. image:: /assets/images/webhook_modal.png
   :width: 100%

Once created, your webhook will begin executing for the chosen events.

******************
 Testing Webhooks
******************

To test a webhook, click on the triple dots on the right of webhook record to expand available
actions.

.. image:: /assets/images/webhook_action.png
   :width: 100%

Clicking on "Test Webhook" will trigger a test event to be sent to the defined webhook URL with a
mock payload as stated below:

.. code::

   {
      "event_id": "b8667b8a-e14d-40e5-83ee-a64e31bdc5f4",
      "event_type": "EXPERIMENT_STATE_CHANGE",
      "timestamp": 1665695871,
      "condition": {
         "state": "COMPLETED"
      },
      "event_data": {
         "data": "test"
      }
   }

*******************
 Deleting Webhooks
*******************

To delete a webhook, click on the triple dots on the right of webhook record to expand available
actions.

.. note::

   We do not support editing webhooks. You can delete and recreate webhooks if needed.

.. toctree::
   :caption: Notification
   :hidden:

   zapier
   slack
