###############
 Through Slack
###############

This section will walk through the steps needed to set up Slack to receive updates from Determined
in a specific Slack channel using Slack Webhook Integrations.

The steps for enabling slack notifications are:

#. :ref:`Creating a Slack Application <slack-app-creation>`
#. :ref:`Enabling Incoming Webhooks in Slack <enabling_webhooks>`
#. :ref:`Configuring Determined for Slack Webhooks <configuring_determined>`
#. :ref:`Setting up the Webhook in Determined <setting-up-webhook-in-determined>`
#. :ref:`Testing the Webhook <testing-webhook>`

.. _slack-app-creation:

******************************
 Creating a Slack Application
******************************

First, we need to create a Slack application and give the application permissions to post in the
appropriate Slack channel. Visit the `Slack App Managment page <https://api.slack.com/apps>`_ and
click on the **Create New App** button.

In app configuration, select **From scratch**.

.. image:: /assets/images/slack-app-configuration.jpeg
   :width: 100%
   :alt: Create an app showing the From Scratch option

In the next window you will input an "App Name" and select the Workspace for the application.

.. _enabling_webhooks:

*************************************
 Enabling Incoming Webhooks in Slack
*************************************

Next, we need to configure incoming webhooks for our Slack application. In your Slack application's
management page navigate to the **Incoming Webhooks** section. Enable the toggle for **Activate
Incoming Webhooks** as shown below.

.. image:: /assets/images/slack-incoming-webhooks-page.jpeg
   :width: 100%
   :alt: Slack API showing the Add New Webhook to Workspace option

Now that webhooks are enabled we can set up a new webhook integration. Click the **Add New Webhook
to Workspace** button at the bottom of the page. On the next page you will be asked to select the
channel that will receive webhook updates. Choose a channel and then press the **Allow** button and
you will be taken back to the Incoming Webhooks page.

.. _configuring_determined:

*******************************************
 Configuring Determined for Slack Webhooks
*******************************************

*Note: The following section is optional but encouraged.*

Determined has the ability to send links to experiments, projects, and workspaces in Slack messages.
To enable Determined to send correctly formatted links you must set the **Base URL** in the
Determined cluster configuration. The **Base URL** is the website address that is used to access the
Determined user interface. The value should be in the format of ``https://yourdomain.com``

There are three ways to set the **Base URL**.

#. Setting a **DET_WEBHOOK_BASE_URL** environment variable.
#. Using the flag ``--webhook-base-url``
#. Adding a ``base_url`` entry to the webhook portion of the master configuration file. An example
   is shown below:

.. code::

   webhook:
       base_url: https://yourdomain.com

If the **Base URL** is set correctly then Slack messages will include links as shown below.

.. image:: /assets/images/slack-message-with-links.png
   :width: 40%
   :alt: Test Webhook Service when Base URL is set correctly

If no **Base URL** is set then links will not be present in Slack messages.

.. image:: /assets/images/slack-message-without-links.png
   :width: 40%
   :alt: Test Webhook Service when no Base URL is set

.. _setting-up-webhook-in-determined:

**************************************
 Setting up the Webhook in Determined
**************************************

Finally, we will need to add a webhook in Determined using **Webhook URL** provided by Slack.

In the **Webhook URLs for Your Workspace** section of Incoming Webhooks page you should see a list
of Webhook URLs for all of the channels that you have added. Click the **Copy** button for the
appropriate Webhook URL and then navigate to the Webhooks page in Determined.

On the Webhooks page in Determined click the **New Webhook** button in the top right corner of the
page.

.. image:: /assets/images/slack-webhook-creation-in-determined.jpeg
   :width: 100%
   :alt: Webhooks page displaying New Webhook fields you will interact with.

In the pop up, paste the **Webhook URL** that was copied from Slack in the **URL** field. Choose
**Slack** for the webhook type and then choose the triggers that you want to receive notifications
for. Finally, select **Create Webhook** and your webhook will be created.

.. _testing-webhook:

*********************
 Testing the Webhook
*********************

To test a Slack webhook in Determined navigate to the Webhooks page and click on the three vertical
dots on the right side of any of the listed webhooks.

.. image:: /assets/images/test-webhook.png
   :width: 100%
   :alt: Webhooks page displaying where to find the Test Webhook action.

If everything has been configured correctly you should receive a message from the Slack application
you created with the message "test" as shown above.
