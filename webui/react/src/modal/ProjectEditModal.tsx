import { Form } from 'antd';
import { useForm, useWatch } from 'antd/lib/form/Form';
import React, { useCallback, useState } from 'react';

import Input from 'components/kit/Input';
import { Modal, Opener } from 'components/kit/Modal';
import { patchProject } from 'services/api';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import handleError from 'utils/error';

interface ProjectModalProps {
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
  }, [form, projectId]);

  return (
    <Modal
      cancelText="No"
      size="medium"
      submit={{
        disabled: submitDisabled,
        handler: handleSubmit,
        onComplete: onComplete,
        text: 'Save Changes',
      }}
      titleText="Edit Project">
      <Form autoComplete="off" form={form} id={FORM_ID} layout="vertical">
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
    </Modal>
  );
};
