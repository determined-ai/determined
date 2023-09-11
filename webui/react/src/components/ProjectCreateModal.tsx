import React, { useCallback } from 'react';

import Form from 'components/kit/Form';
import Input from 'components/kit/Input';
import { Modal } from 'components/kit/Modal';
import { paths } from 'routes/utils';
import { createProject } from 'services/api';
import handleError, { DetError, ErrorLevel, ErrorType } from 'utils/error';
import { routeToReactUrl } from 'utils/routes';

const FORM_ID = 'create-project-form';

interface FormInputs {
  description?: string;
  projectName: string;
}

interface Props {
  onClose?: () => void;
  workspaceId: number;
}

const ProjectCreateModalComponent: React.FC<Props> = ({ onClose, workspaceId }: Props) => {
  const [form] = Form.useForm<FormInputs>();
  const projectName = Form.useWatch('projectName', form);

  const handleSubmit = useCallback(async () => {
    const values = await form.validateFields();

    try {
      if (values) {
        const response = await createProject({
          description: values.description,
          name: values.projectName,
          workspaceId,
        });
        routeToReactUrl(paths.projectDetails(response.id));
        form.resetFields();
      }
    } catch (e) {
      if (e instanceof DetError) {
        handleError(e, {
          level: e.level,
          publicMessage: e.publicMessage,
          publicSubject: 'Unable to create project.',
          silent: false,
          type: e.type,
        });
      } else {
        handleError(e, {
          level: ErrorLevel.Error,
          publicMessage: 'Please try again later.',
          publicSubject: 'Unable to create project.',
          silent: false,
          type: ErrorType.Server,
        });
      }
    }
  }, [form, workspaceId]);

  return (
    <Modal
      cancel
      size="small"
      submit={{
        disabled: !projectName,
        form: FORM_ID,
        handleError,
        handler: handleSubmit,
        text: 'Create Project',
      }}
      title="New Project"
      onClose={onClose}>
      <Form autoComplete="off" form={form} id={FORM_ID} layout="vertical">
        <Form.Item
          label="Project Name"
          name="projectName"
          rules={[{ message: 'Name is required', required: true }]}>
          <Input maxLength={80} />
        </Form.Item>
        <Form.Item label="Description" name="description">
          <Input />
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default ProjectCreateModalComponent;
