import Form from 'hew/Form';
import Input from 'hew/Input';
import { Modal } from 'hew/Modal';
import Select, { Option } from 'hew/Select';
import { useToast } from 'hew/Toast';
import React, { useCallback, useEffect, useId, useMemo, useState } from 'react';

import useFeature from 'hooks/useFeature';
import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';
import { createWebhook } from 'services/api';
import { V1TriggerType, V1WebhookMode, V1WebhookType } from 'services/api-ts-sdk/api';
import workspaceStore from 'stores/workspaces';
import { RunState, Workspace } from 'types';
import handleError, { DetError, ErrorLevel, ErrorType } from 'utils/error';
import { useObservable } from 'utils/observable';
import { routeToReactUrl } from 'utils/routes';
interface Props {
  onSuccess?: () => void;
}

const FORM_ID = 'create-webhook-form';

const triggerEvents = [
  RunState.Completed,
  RunState.Error,
  V1TriggerType.TASKLOG,
  V1TriggerType.CUSTOM,
] as const;

interface FormInputs {
  regex?: string;
  triggerEvents: (typeof triggerEvents)[number][];
  url: string;
  webhookType: V1WebhookType;
  name: string;
  workspaceId: number;
  mode: V1WebhookMode;
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
  {
    label: 'CUSTOM',
    value: V1TriggerType.CUSTOM,
  },
];
const modeOptions = [
  {
    label: 'All experiments in Workspace',
    value: V1WebhookMode.WORKSPACE,
  },
  {
    label: 'Specific experiment(s) with matching configuration',
    value: V1WebhookMode.SPECIFIC,
  },
];
const WebhookCreateModalComponent: React.FC<Props> = ({ onSuccess }: Props) => {
  const idPrefix = useId();
  const [form] = Form.useForm<FormInputs>();
  const [disabled, setDisabled] = useState<boolean>(true);
  const triggers = Form.useWatch('triggerEvents', form);
  const f_webhook = useFeature().isOn('webhook_improvement');
  const workspaces = useObservable(workspaceStore.workspaces).getOrElse([]);
  const { canCreateWebhooks } = usePermissions();
  const { openToast } = useToast();

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

  const permWorkspace = useMemo(
    () => canCreateWebhooks(workspaces),
    [workspaces, canCreateWebhooks],
  );

  const handleSubmit = useCallback(async () => {
    const values = await form.validateFields();

    try {
      if (values) {
        await createWebhook({
          mode: values.mode,
          name: values.name,
          triggers: values.triggerEvents.map((state) => {
            if (state === V1TriggerType.TASKLOG) {
              return {
                condition: { regex: values.regex },
                triggerType: state,
              };
            }
            if (state === V1TriggerType.CUSTOM) {
              return {
                condition: {},
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
          workspaceId: values.workspaceId,
        });
        onSuccess?.();
        routeToReactUrl(paths.webhooks());
        form.resetFields();
        openToast({
          severity: 'Confirm',
          title: 'Webhook created.',
        });
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
      throw e;
    }
  }, [form, onSuccess, openToast]);

  useEffect(() => {
    if ((triggers || []).includes(V1TriggerType.CUSTOM)) {
      form.setFieldValue('mode', V1WebhookMode.SPECIFIC);
    }
  }, [triggers, form]);

  return (
    <Modal
      cancel
      size="medium"
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
        {f_webhook && (
          <>
            <Form.Item
              label="Workspace"
              name="workspaceId"
              rules={[{ message: 'Workspace is required', required: true }]}>
              <Select allowClear placeholder="Workspace (required)">
                {permWorkspace.map((workspace: Workspace) => (
                  <Option key={workspace.id} value={workspace.id}>
                    {workspace.name}
                  </Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item
              label="Name"
              name="name"
              rules={[{ message: 'Name is required.', required: true }]}>
              <Input />
            </Form.Item>
          </>
        )}
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
          initialValue={V1WebhookType.DEFAULT}
          label="Type"
          name="webhookType"
          rules={[{ message: 'Webhook type is required ', required: true }]}>
          <Select options={typeOptions} placeholder="Select type of Webhook" />
        </Form.Item>
        <Form.Item
          label="Trigger"
          name="triggerEvents"
          rules={[{ message: 'At least one trigger event is required', required: true }]}>
          <Select
            mode="multiple"
            options={f_webhook ? triggerOptions : triggerOptions.slice(0, 3)}
            placeholder="Select trigger event"
          />
        </Form.Item>
        {(triggers || []).includes(V1TriggerType.TASKLOG) && (
          <Form.Item
            label="Regex"
            name="regex"
            rules={[{ message: 'Regex is required when triggering on task log', required: true }]}>
            <Input />
          </Form.Item>
        )}
        {f_webhook && (
          <Form.Item
            initialValue={V1WebhookMode.WORKSPACE}
            label="Trigger by"
            name="mode"
            rules={[{ message: 'Webhook mode is required', required: true }]}>
            <Select
              disabled={(triggers || []).includes(V1TriggerType.CUSTOM)}
              options={modeOptions}
              placeholder="Select webhook mode"
            />
          </Form.Item>
        )}
      </Form>
    </Modal>
  );
};

export default WebhookCreateModalComponent;
