import Alert from 'hew/Alert';
import CodeEditor from 'hew/CodeEditor';
import Form from 'hew/Form';
import Input from 'hew/Input';
import { Modal } from 'hew/Modal';
import Select, { Option } from 'hew/Select';
import { useToast } from 'hew/Toast';
import { Loadable } from 'hew/utils/loadable';
import yaml from 'js-yaml';
import { useObservable } from 'micro-observables';
import React, { useCallback, useId, useState } from 'react';

import { createTaskTemplate } from 'services/api';
import workspaceStore from 'stores/workspaces';
import { Workspace } from 'types';
import handleError, { DetError, ErrorLevel, ErrorType } from 'utils/error';

const FORM_ID = 'create-template-form';

interface FormInputs {
  name: string;
  workspaceId: number;
  config: string;
}

interface Props {
  workspaceId?: number;
}

const TemplateCreateModalComponent: React.FC<Props> = ({ workspaceId }) => {
  const idPrefix = useId();
  const { openToast } = useToast();
  const [form] = Form.useForm<FormInputs>();
  const [disabled, setDisabled] = useState<boolean>(true);
  const workspaces = Loadable.getOrElse([], useObservable(workspaceStore.workspaces));

  const onChange = useCallback(() => {
    const fields = form.getFieldsError();
    const hasError = fields.some((f) => f.errors.length);
    setDisabled(hasError);
  }, [form]);

  const handleSubmit = useCallback(async () => {
    const values = await form.validateFields();

    try {
      if (values) {
        await createTaskTemplate({
          ...values,
          config: yaml.load(values.config),
        });
        form.resetFields();
        openToast({
          description: `Template ${values.name} has been created`,
          severity: 'Info',
          title: 'Template Created',
        });
      }
    } catch (e) {
      if (e instanceof DetError) {
        handleError(e, {
          level: e.level,
          publicMessage: e.publicMessage,
          publicSubject: 'Unable to create template.',
          silent: false,
          type: e.type,
        });
      } else {
        handleError(e, {
          level: ErrorLevel.Error,
          publicMessage: 'Please try again later.',
          publicSubject: 'Unable to create template.',
          silent: false,
          type: ErrorType.Server,
        });
      }
    }
  }, [form, openToast]);

  return (
    <Modal
      cancel
      size="small"
      submit={{
        disabled,
        form: idPrefix + FORM_ID,
        handleError,
        handler: handleSubmit,
        text: 'Create Template',
      }}
      title="New Template">
      <Form
        autoComplete="off"
        form={form}
        id={idPrefix + FORM_ID}
        layout="vertical"
        onFieldsChange={onChange}>
        <Form.Item
          label="Name"
          name="name"
          rules={[{ message: 'Name is required.', required: true }]}>
          <Input />
        </Form.Item>
        <Form.Item
          initialValue={workspaceId}
          label="Workspace"
          name="workspaceId"
          rules={[{ message: 'Workspace is required', required: true, type: 'number' }]}>
          <Select allowClear disabled={!!workspaceId} placeholder="Workspace (required)">
            {workspaces.map((workspace: Workspace) => (
              <Option key={workspace.id} value={workspace.id}>
                {workspace.name}
              </Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item
          label="Config"
          name="config"
          rules={[
            { message: 'Template content is required', required: true },
            {
              validator: (_rule, value) => {
                try {
                  yaml.load(value);
                  return Promise.resolve();
                } catch (err: unknown) {
                  return Promise.reject(
                    new Error(
                      `Invalid YAML on line ${(err as { mark: { line: string } }).mark.line}.`,
                    ),
                  );
                }
              },
            },
          ]}>
          <CodeEditor
            file={''}
            files={[{ key: 'template.yaml' }]}
            height="40vh"
            onError={handleError}
          />
        </Form.Item>
        {form.getFieldError('config').length > 0 && (
          <Alert message={form.getFieldError('config').join('\n')} type="error" />
        )}
      </Form>
    </Modal>
  );
};

export default TemplateCreateModalComponent;
