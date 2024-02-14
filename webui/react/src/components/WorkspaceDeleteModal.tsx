import Form from 'hew/Form';
import Input from 'hew/Input';
import { Modal } from 'hew/Modal';
import React, { useCallback, useId } from 'react';

import { paths } from 'routes/utils';
import workspaceStore from 'stores/workspaces';
import { Workspace } from 'types';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';
import { routeToReactUrl } from 'utils/routes';

import css from './WorkspaceDeleteModal.module.scss';

const FORM_ID = 'delete-workspace-form';

interface FormInputs {
  workspaceName: string;
}

interface Props {
  onClose?: () => void;
  returnIndexOnDelete: boolean;
  workspace: Workspace;
}

const WorkspaceDeleteModalComponent: React.FC<Props> = ({
  onClose,
  returnIndexOnDelete,
  workspace,
}: Props) => {
  const idPrefix = useId();
  const [form] = Form.useForm<FormInputs>();
  const workspaceNameValue = Form.useWatch('workspaceName', form);

  const handleSubmit = useCallback(async () => {
    try {
      await workspaceStore.deleteWorkspace(workspace.id);
      if (returnIndexOnDelete) {
        routeToReactUrl(paths.workspaceList());
      }
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicSubject: 'Unable to delete workspace.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [workspace.id, returnIndexOnDelete]);

  return (
    <Modal
      cancel
      danger
      size="small"
      submit={{
        disabled: workspaceNameValue !== workspace.name,
        form: idPrefix + FORM_ID,
        handleError,
        handler: handleSubmit,
        text: 'Delete Workspace',
      }}
      title="Delete Workspace"
      onClose={onClose}>
      <Form autoComplete="off" form={form} id={idPrefix + FORM_ID} layout="vertical">
        <p>
          Are you sure you want to delete{' '}
          <strong className={css.workspaceName}>&quot;{workspace.name}&quot;</strong>?
        </p>
        <p>
          All projects, experiments, and notes within it will also be deleted. This cannot be
          undone.
        </p>
        <Form.Item
          label={
            <div>
              Please type <mark className={css.workspaceName}>{workspace.name}</mark> to confirm
            </div>
          }
          name="workspaceName"
          rules={[
            {
              message: 'Please type the workspace name to confirm',
              pattern: new RegExp(`^${workspace.name}$`),
              required: true,
            },
          ]}>
          <Input autoComplete="off" />
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default WorkspaceDeleteModalComponent;
