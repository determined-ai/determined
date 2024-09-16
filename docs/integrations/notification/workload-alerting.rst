.. _workload-alerting:

###################
 Workload Alerting
###################

Workload alerting allows you to monitor the state of your experiments and share important information with your team members. This feature enables proactive issue detection while maintaining a good signal-to-noise ratio.

.. note::
   This is an experimental feature. You need to enable the "Webhook Improvement" feature to use these capabilities.

*****************
 Key Concepts
*****************

- Webhook Subscription Modes: "Open" (All experiments in the Workspace) and "Run specific" (Specific experiment(s) with matching configuration)
- Webhook Exclusion
- Custom Triggers
- Alert Levels: INFO, WARN, DEBUG, ERROR

*******************
 Creating Webhooks
*******************

As a non-admin user, you can configure webhooks. Here's how to create webhooks in different workspaces:

1. Navigate to the Webhooks section in the WebUI.
2. Click "New Webhook".
3. In the New Webhook dialogue:
   - Select your Workspace
   - Name your webhook
   - Paste the webhook URL (e.g., from Zapier)
   - Set type to Default
   - Select the trigger event (COMPLETED, ERROR, TASKLOG, or CUSTOM)
   - In "Trigger by", select either "All experiments in the Workspace" or "Specific experiment(s) with matching configuration"
4. Click Create Webhook

*******************
 Use Cases
*******************

Determined supports global and workspace-specific webhooks.

Global Webhooks
===============

Global webhooks are triggered for all experiments across all workspaces, providing a centralized way to monitor and receive notifications for your entire cluster.

To create a global webhook:

1. Navigate to the Webhooks section in the WebUI.
2. Click "New Webhook".
3. In the New Webhook dialogue:
   - Leave the Workspace field empty or select "Global" if available
   - Follow the same steps as creating a regular webhook for the remaining fields

Global webhooks are particularly useful for cluster-wide monitoring and alerting. They can be used to:

- Track overall cluster usage and performance
- Monitor for system-wide issues or errors
- Provide a comprehensive view of all experiments running across different workspaces

.. note::
   Global webhooks require appropriate permissions. If you don't see the option to create a global webhook, consult with your system administrator.

Triggering Webhooks Within the Same Workspace
=============================================

Experiments only trigger webhooks within the same workspace. To verify:

1. Create an experiment in the workspace where you set up the webhook.
2. Run the experiment.
3. Check the event log in your webhook service (e.g., Zapier) to see if it was triggered upon experiment completion.

Triggering Specific Webhooks with Matching Configurations
=========================================================

To set up a webhook that only triggers for specific experiments:

1. Create a webhook with the "Run specific" mode.
2. Set "Trigger by" to "Specific experiment(s) with matching configuration".
3. In the experiment configuration, reference the webhook:

   .. code:: yaml

      integrations:
        webhooks:
          webhook_name:
            - your-specific-webhook-name

4. Run the experiment and check the event log in your webhook service.

Excluding Experiments from Webhooks
===================================

To prevent a specific experiment from triggering webhooks:

1. Edit the experiment configuration:

   .. code:: yaml

      integrations:
        webhooks:
          exclude: true

2. Run the experiment and verify that no webhooks are triggered.

Using Custom Triggers
=====================

To create and use custom triggers:

1. Create a new webhook with the trigger set to "CUSTOM".
2. Edit the experiment config to match the custom trigger:

   .. code:: yaml

      integrations:
        webhooks:
          webhook_name:
            - your-custom-webhook-name

3. In your experiment code, use the `core_context.alert()` function to trigger the webhook:

   .. code:: python

      core_context.alert(
          title="Custom Alert",
          description="This is a custom alert",
          level="INFO"
      )

4. Run the experiment and check the event log in your webhook service for the custom data.

*****************
 Best Practices
*****************

- Use specific webhooks for critical experiments to avoid alert fatigue.
- Leverage custom triggers for fine-grained control over when alerts are sent.
- Regularly review and update your webhook configurations to ensure they remain relevant and useful.

