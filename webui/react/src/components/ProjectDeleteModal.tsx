import React, { useCallback } from 'react';

import Form from 'components/kit/Form';
import Input from 'components/kit/Input';
import { Modal } from 'components/kit/Modal';
import { deleteProject } from 'services/api';
import { Project } from 'types';
import { ErrorLevel, ErrorType } from 'utils/error';
import handleError from 'utils/error';

interface FormInputs {
  projectName: string;
}

interface Props {
  onClose?: () => void;
  onDelete?: () => void;
  project: Project;
}

const ProjectDeleteModalComponent: React.FC<Props> = ({ onClose, project, onDelete }: Props) => {
  const [form] = Form.useForm<FormInputs>();
  const projectNameValue = Form.useWatch('projectName', form);

  const handleSubmit = useCallback(async () => {
    try {
      await deleteProject({ id: project.id });
      if (onDelete) onDelete();
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to delete project.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [project.id, onDelete]);

  return (
    <Modal
      cancel
      danger
      size="small"
      submit={{
        disabled: projectNameValue !== project.name,
        handleError,
        handler: handleSubmit,
        text: 'Delete Project',
      }}
      title="Delete Project"
      onClose={onClose}>
      <Form autoComplete="off" form={form} layout="vertical">
        <p>
          Are you sure you want to delete <strong>&quot;{project.name}&quot;</strong>?
        </p>
        <p>All experiments and notes within it will also be deleted. This cannot be undone.</p>
        <Form.Item
          label={
            <div>
              Please type <strong>{project.name}</strong> to confirm
            </div>
          }
          name="projectName"
          rules={[
            {
              message: 'Please type the project name to confirm',
              pattern: new RegExp(`^${project.name}$`),
              required: true,
            },
          ]}>
          <Input autoComplete="off" />
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default ProjectDeleteModalComponent;
