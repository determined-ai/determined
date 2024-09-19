.. _workload-alerting:

###################
 Workload Alerting
###################

Workload alerting allows you to monitor the state of your experiments and share important
information with your team members. This feature enables proactive issue detection while maintaining
a good signal-to-noise ratio.

.. note::

   To use this experimental feature, enable "Webhook Improvement" in :ref:`user settings
   <web-ui-user-settings>`.

**************
 Key Concepts
**************

-  Webhook Trigger options: "All experiments in Workspace" and "Specific experiment(s) with matching
   configuration"
-  Webhook Exclusion
-  Trigger Types: COMPLETED, ERROR, TASKLOG, CUSTOM
-  Alert Levels: INFO, WARN, DEBUG, ERROR

For detailed information on supported triggers and example usage, see :ref:`notifications`.

.. _creating-webhooks:

*******************
 Creating Webhooks
*******************

As a non-admin user with Editor or higher permissions, you can configure webhooks within your
workspace. Here's how to create webhooks:

#. Navigate to the **Webhooks** section in the WebUI.

#. Select **New Webhook**.

#. In the New Webhook dialogue:

   -  Select your Workspace
   -  Name your webhook
   -  Paste the webhook URL (e.g., from Zapier)
   -  Set Type to either Default or Slack
   -  Select the Trigger event (COMPLETED, ERROR, TASKLOG, or CUSTOM)
   -  Choose the Trigger by option: "All experiments in Workspace" or "Specific experiment(s) with
      matching configuration"
   -  If "Specific experiment(s) with matching configuration", note the Webhook Name for use in
      experiment configurations

#. Click **Create Webhook**.

*******************
 Deleting Webhooks
*******************

To delete a webhook, select the more-options menu to the right of the webhook record to expand
available actions.

******************
 Editing Webhooks
******************

To edit a webhook, select the more-options menu to the right of the webhook record to expand
available actions.

.. note::

   Determined only supports editing the URL of webhooks. To modify other attributes, delete and
   recreate the webhook.

***********
 Use Cases
***********

Webhooks in Determined offer versatile solutions for various monitoring and alerting needs. Let's
explore some common use cases to help you leverage this powerful feature effectively.

Case 1: Share Simple State on All Experiments in Workspace
==========================================================

This use case is ideal for teams that want to maintain a broad overview of all experiments running
in a workspace, ensuring that no important updates are missed.

#. Create a webhook with the "All experiments in Workspace" option.
#. Select the desired trigger events (COMPLETED, ERROR, TASKLOG).
#. All experiments in the workspace will now trigger this webhook unless explicitly excluded.

Case 2: Exclude Specific Experiments from Triggering Webhooks
=============================================================

During active development or debugging, you may want to prevent certain experiments from triggering
alerts to reduce noise and focus on specific tasks.

#. Edit the experiment configuration:

   .. code:: yaml

      integrations:
        webhooks:
          exclude: true

#. Run the experiment and verify that no webhooks are triggered.

Case 3: Customizable Monitoring for Specific Experiments
========================================================

For critical experiments or those requiring special attention, you can set up custom monitoring to
receive tailored alerts based on specific conditions or events in your code.

#. Create a webhook with the "Specific experiment(s) with matching configuration" option and
   "CUSTOM" trigger type.

#. Note the Webhook Name.

#. In the experiment configuration, reference the webhook:

   .. code:: yaml

      integrations:
        webhooks:
          webhook_name:
            - <webhook_name>

#. In your experiment code, use the `core_context.alert()` function to trigger the webhook:

   .. code:: python

      with det.core.init() as core_context:
          core_context.alert(
              title="Custom Alert",
              description="This is a custom alert",
              level="INFO"
          )

#. Run the experiment and check the event log in your webhook service for the custom data.

For more details on custom triggers, see :ref:`notifications`.

****************
 Best Practices
****************

-  Use "Open" subscription mode for general monitoring of all experiments in a workspace.
-  Leverage "Run specific" mode and custom triggers for fine-grained control over alerts for
   critical experiments.
-  Use webhook exclusion for experiments under active iteration to reduce noise.
-  Regularly review and update your webhook configurations to ensure they remain relevant and
   useful.
