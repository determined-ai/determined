################
 Through Zapier
################

This section will walk through the steps needed to set up Zapier webhook to to receive updates from
Determined.

The steps to set up Zapier webhook are:

#. :ref:`Creating a Zap with Webhook <zap-creation>`
#. :ref:`Setting up the Webhook in Determined <webhook-in-determined>`
#. :ref:`Testing the Webhook <testing-webhook-zapier>`
#. :ref:`Verifing the Signature <verification>`
#. :ref:`Configuring Destination <configuring_destination>`

.. _zap-creation:

*****************************
 Creating a Zap with Webhook
*****************************

First, you need to create a Zap with webhook. Visit the `Zapier Website
<https://zapier.com/app/zaps>`_, signup if you haven't already, and click on the **Create Zap**
button.

Select **Webhooks by Zapier** as trigger **Catch Raw Hook** as event. Using **Catch Raw Hook**
intead of **Catch Hook** because headers are needed to verify each webhook request.

.. note::

   You need to upgrade to premium account to access **Webhooks by Zapier**

.. image:: /assets/images/zapier_webhook.png
   :width: 100%
   :alt: Zapier Webhooks page displaying Catch Raw Hook event selected

.. _webhook-in-determined:

**************************************
 Setting up the Webhook in Determined
**************************************

Then, you need to create a webhook in Determined using the **Webhook URL** from Zapier.

.. image:: /assets/images/zapier_webhook_url.png
   :width: 100%
   :alt: This is where your webhook URL displays in Zapier

Navigate to ``/det/webhooks`` or click on the "Webhooks" item in navigation side menu, then click
the **New Webhook** button in the top right corner of the page.

.. image:: /assets/images/zapier_new_webhook.png
   :width: 100%
   :alt: Webhooks page displaying New Webhook fields including triggers.

Paste the **Webhook URL** that was copied from Zapier in the **URL** field. Select **Default** for
the webhook type and then select the triggers that you want to receive notifications for. Finally,
select **Create Webhook** and your webhook will be created.

.. _testing-webhook-zapier:

*********************
 Testing the Webhook
*********************

To send a test payload, click on the triple dots on the right of webhook record and click on **Test
Webhook**.

.. image:: /assets/images/zapier_test.png
   :width: 100%
   :alt: Webhooks page displaying where to find the Test Webhook action.

Then navigate back to Zapier and click on **Test Trigger**, then you should be able to see the test
request.

.. image:: /assets/images/zapier_request_found.png
   :width: 100%
   :alt: Zapier Webhooks request page showing that your request was found.

.. _verification:

************************
 Verifing the Signature
************************

Refer to :ref:`Security and Signed Payload <webhook_security>` for the details behind verifing
signature.

In Zapier, you can use **Code by Zapier** to compute signature based on payload and timestamp, then
compare it with the signature in the request to verify each request.

Add a new action and choose **Code by Zapier**, select **Run Python** as an example.

.. image:: /assets/images/zapier_python.png
   :width: 100%
   :alt: Code by Zapier action with a Run Python event

Construct input data as following:

-  webhook_signing_key: match the ``webhook_signing_key`` in Determined.
-  timestamp: ``X-Determined-AI-Signature-Timestamp`` from request header.
-  signature: ``X-Determined-AI-Signature`` from request header.
-  payload: raw body of request.

.. image:: /assets/images/zapier_code_input.png
   :width: 100%
   :alt: Code by Zapier showing set up action input data like webhook_signing_key

Input code as following:

.. code::

   import hmac, hashlib, json

   signing_key = input_data['webhook_signing_key']
   timestamp = input_data['timestamp']
   signature = input_data['signature']
   payload = json.loads(input_data['payload'])

   calculated_signature = hmac.new(signing_key.encode(), f"{timestamp},{payload}".encode(), digestmod=hashlib.sha256).hexdigest()

   if calculated_signature == signature:
       return {"result": "PASS", "payload": payload}
   return {"result": "Signature cannot be verified, request might not be legit"}

Under **Test Action**, test the code above, you should be able to see that verification has passed.

.. image:: /assets/images/zapier_code_result.png
   :width: 100%
   :alt: Code by Zapier showing that a Run Python event was sent

.. _configuring_destination:

*************************
 Configuring Destination
*************************

Finally, you can configure where to proceed under each scenario by adding more actions. For example,
send out an alert when verification fails, or send out an email with experiment information when
verification pass.
