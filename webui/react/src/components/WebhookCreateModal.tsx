import Form from 'hew/Form';
import Input from 'hew/Input';
import { Modal } from 'hew/Modal';
import Select from 'hew/Select';
import React, { useCallback, useEffect, useId, useState } from 'react';

import { paths } from 'routes/utils';
import { createWebhook } from 'services/api';
import { V1TriggerType, V1WebhookType } from 'services/api-ts-sdk/api';
import { RunState } from 'types';
import handleError, { DetError, ErrorLevel, ErrorType } from 'utils/error';
import { routeToReactUrl } from 'utils/routes';

interface Props {
  onSuccess?: () => void;
}

const FORM_ID = 'create-webhook-form';

const triggerEvents = [RunState.Completed, RunState.Error, V1TriggerType.TASKLOG] as const;

interface FormInputs {
  regex?: string;
  triggerEvents: (typeof triggerEvents)[number][];
  url: string;
  webhookType: V1WebhookType;
}

const typeOptions = [
  {
    label: 'Default',
    value: V1WebhookType.DEFAULT,
  },
  {
    label: 'Slack',
    value: V1WebhookType.SLACK,
  },
];
const triggerOptions = [
  {
    label: RunState.Completed,
    value: RunState.Completed,
  },
  {
    label: RunState.Error,
    value: RunState.Error,
  },
  {
    label: 'TASKLOG',
    value: V1TriggerType.TASKLOG,
  },
];
const WebhookCreateModalComponent: React.FC<Props> = ({ onSuccess }: Props) => {
  const idPrefix = useId();
  const [form] = Form.useForm<FormInputs>();
  const [disabled, setDisabled] = useState<boolean>(true);
  const triggers = Form.useWatch('triggerEvents', form);

  const onChange = useCallback(() => {
    const fields = form.getFieldsError();
    const hasError = fields.some((f) => f.errors.length);
    setDisabled(hasError);
  }, [form]);

  useEffect(() => {
    if (!(triggers || []).includes('TRIGGER_TYPE_TASK_LOG')) {
      form.setFieldValue('regex', null);
    }
  }, [triggers, form]);

  const handleSubmit = useCallback(async () => {
    const values = await form.validateFields();

    try {
      if (values) {
        await createWebhook({
          triggers: values.triggerEvents.map((state) => {
            if (state === 'TRIGGER_TYPE_TASK_LOG') {
              return {
                condition: { regex: values.regex },
                triggerType: state,
              };
            }
            return {
              condition: { state },
              triggerType: V1TriggerType.EXPERIMENTSTATECHANGE,
            };
          }),
          url: values.url,
          webhookType: values.webhookType,
        });
        onSuccess?.();
        routeToReactUrl(paths.webhooks());
        form.resetFields();
      }
    } catch (e) {
      if (e instanceof DetError) {
        handleError(e, {
          level: e.level,
          publicMessage: e.publicMessage,
          publicSubject: 'Unable to create webhook.',
          silent: false,
          type: e.type,
        });
      } else {
        handleError(e, {
          level: ErrorLevel.Error,
          publicMessage: 'Please try again later.',
          publicSubject: 'Unable to create webhook.',
          silent: false,
          type: ErrorType.Server,
        });
      }
    }
  }, [form, onSuccess]);

  return (
    <Modal
      cancel
      size="small"
      submit={{
        disabled,
        form: idPrefix + FORM_ID,
        handleError,
        handler: handleSubmit,
        text: 'Create Webhook',
      }}
      title="New Webhook">
      <Form
        autoComplete="off"
        form={form}
        id={idPrefix + FORM_ID}
        layout="vertical"
        onFieldsChange={onChange}>
        <Form.Item
          label="URL"
          name="url"
          rules={[
            { message: 'URL is required.', required: true },
            { message: 'URL must be valid.', type: 'url', whitespace: true },
          ]}>
          <Input />
        </Form.Item>
        <Form.Item
          label="Type"
          name="webhookType"
          rules={[{ message: 'Webhook type is required ', required: true }]}>
          <Select options={typeOptions} placeholder="Select type of Webhook" />
        </Form.Item>
        <Form.Item
          label="Trigger"
          name="triggerEvents"
          rules={[{ message: 'At least one trigger event is required', required: true }]}>
          <Select mode="multiple" options={triggerOptions} placeholder="Select trigger event" />
        </Form.Item>
        {(triggers || []).includes(V1TriggerType.TASKLOG) && (
          <Form.Item
            label="Regex"
            name="regex"
            rules={[{ message: 'Regex is required when triggering on task log', required: true }]}>
            <Input />
          </Form.Item>
        )}
      </Form>
    </Modal>
  );
};

export default WebhookCreateModalComponent;
