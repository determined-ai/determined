import Form from 'hew/Form';
import Input from 'hew/Input';
import { Modal } from 'hew/Modal';
import { escapeRegExp } from 'lodash';
import React, { useCallback, useId } from 'react';

import { deleteProject } from 'services/api';
import { Project } from 'types';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';

const FORM_ID = 'delete-project-form';

interface FormInputs {
  projectName: string;
}

interface Props {
  onClose?: () => void;
  onDelete?: () => void;
  project: Project;
}

const ProjectDeleteModalComponent: React.FC<Props> = ({ onClose, project, onDelete }: Props) => {
  const idPrefix = useId();
  const [form] = Form.useForm<FormInputs>();
  const projectNameValue = Form.useWatch('projectName', form);

  const handleSubmit = useCallback(async () => {
    try {
      await deleteProject({ id: project.id });
      onDelete?.();
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
        form: idPrefix + FORM_ID,
        handleError,
        handler: handleSubmit,
        text: 'Delete Project',
      }}
      title="Delete Project"
      onClose={onClose}>
      <Form autoComplete="off" form={form} id={idPrefix + FORM_ID} layout="vertical">
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
              pattern: new RegExp(`^${escapeRegExp(project.name)}$`),
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
