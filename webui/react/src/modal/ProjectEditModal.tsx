import { Form } from 'antd';
import React, { useCallback, useMemo } from 'react';

import { useForm, useWatch } from 'antd/lib/form/Form';
import Input from 'components/kit/Input';
import { patchProject } from 'services/api';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import handleError from 'utils/error';
import { useModalParams } from './useModality';
import css from './useModality.module.scss';

interface ProjectModalProps extends JSX.IntrinsicAttributes {
  projectId: number;
  initialName: string;
  initialDescription: string;
  onComplete: () => Promise<void>;
}
interface FormInputs {
  description: string;
  name: string;
  projectId: number;
}

const FORM_ID = 'edit-project-form';
export const ProjectModal: React.FC<ProjectModalProps> = ({
  projectId,
  initialName,
  initialDescription,
  onComplete,
}) => {
  const [form] = useForm<FormInputs>();

  const projectName = useWatch('name', form);
  const submitDisabled = !projectName;

  const handleSubmit = useCallback(async () => {
    const { name, description } = await form.validateFields(['name', 'description']);

    try {
      await patchProject({ description, id: projectId, name });
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to edit project.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [form]);

  const params = useMemo(
    () => ({
      cancelText: 'No',
      size: 'medium' as const,
      submit: {
        disabled: submitDisabled,
        handler: handleSubmit,
        onComplete: onComplete,
        text: 'Save Changes',
      },
      titleText: 'Edit Project',
    }),
    [handleSubmit, submitDisabled],
  );

  useModalParams(params);

  return (
    <Form autoComplete="off" className={css.base} form={form} id={FORM_ID} layout="vertical">
      <Form.Item
        initialValue={initialName}
        label="Project Name"
        name="name"
        rules={[{ message: 'Project name is required', required: true }]}>
        <Input maxLength={80} />
      </Form.Item>
      <Form.Item initialValue={initialDescription} label="Description" name="description">
        <Input />
      </Form.Item>
    </Form>
  );
};
