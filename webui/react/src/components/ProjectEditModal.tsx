import React, { useCallback } from 'react';

import Form from 'components/kit/Form';
import Input from 'components/kit/Input';
import { Modal } from 'components/kit/Modal';
import { patchProject } from 'services/api';
import { Project } from 'types';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';

const FORM_ID = 'edit-project-form';

interface FormInputs {
  description?: string;
  projectName: string;
}

interface Props {
  onEdit?: () => void;
  project: Project;
}

const ProjectEditModalComponent: React.FC<Props> = ({ onEdit, project }: Props) => {
  const [form] = Form.useForm<FormInputs>();
  const projectName = Form.useWatch('projectName', form);

  const handleSubmit = useCallback(async () => {
    const values = await form.validateFields();
    const name = values.projectName;
    const description = values.description;

    try {
      await patchProject({ description, id: project.id, name });
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to edit project.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [form, project.id]);

  return (
    <Modal
      cancel
      size="small"
      submit={{
        disabled: !projectName,
        handleError,
        handler: handleSubmit,
        text: 'Save Changes',
      }}
      title="Edit Project"
      onClose={() => {
        form.resetFields();
        onEdit?.();
      }}>
      <Form autoComplete="off" form={form} id={FORM_ID} layout="vertical">
        <Form.Item
          initialValue={project.name}
          label="Project Name"
          name="projectName"
          rules={[{ message: 'Project name is required', required: true }]}>
          <Input maxLength={80} />
        </Form.Item>
        <Form.Item initialValue={project.description} label="Description" name="description">
          <Input />
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default ProjectEditModalComponent;
