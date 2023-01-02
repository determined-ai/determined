import { Form, Input } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { useCallback, useEffect, useMemo } from 'react';

import { paths } from 'routes/utils';
import useModal, { ModalHooks } from 'shared/hooks/useModal/useModal';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { routeToReactUrl } from 'shared/utils/routes';
import { useDeleteWorkspace } from 'stores/workspaces';
import { Workspace } from 'types';
import handleError from 'utils/error';

import css from './useModalWorkspaceDelete.module.scss';

interface FormInputs {
  workspaceName: string;
}

interface Props {
  onClose?: () => void;
  returnIndexOnDelete: boolean;
  workspace: Workspace;
}

const useModalWorkspaceDelete = ({
  onClose,
  returnIndexOnDelete,
  workspace,
}: Props): ModalHooks => {
  const [form] = Form.useForm<FormInputs>();
  const workspaceNameValue = Form.useWatch('workspaceName', form);

  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal({ onClose });
  const deleteWorkspace = useDeleteWorkspace();

  const modalContent = useMemo(() => {
    return (
      <Form autoComplete="off" className={css.base} form={form} layout="vertical">
        <p>
          Are you sure you want to delete <strong>&quot;{workspace.name}&quot;</strong>?
        </p>
        <p>
          All projects, experiments, and notes within it will also be deleted. This cannot be
          undone.
        </p>
        <Form.Item
          label={
            <div>
              Please type <strong>{workspace.name}</strong> to confirm
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
    );
  }, [form, workspace.name]);

  const handleOk = useCallback(async () => {
    try {
      await deleteWorkspace(workspace.id);
      if (returnIndexOnDelete) {
        routeToReactUrl(paths.workspaceList());
      }
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to delete workspace.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [workspace.id, deleteWorkspace, returnIndexOnDelete]);

  const getModalProps = useCallback((): ModalFuncProps => {
    return {
      closable: true,
      content: modalContent,
      icon: null,
      okButtonProps: { danger: true, disabled: workspaceNameValue !== workspace.name },
      okText: 'Delete Workspace',
      onOk: handleOk,
      title: 'Delete Workspace',
    };
  }, [handleOk, modalContent, workspace.name, workspaceNameValue]);

  const modalOpen = useCallback(
    (initialModalProps: ModalFuncProps = {}) => {
      openOrUpdate({ ...getModalProps(), ...initialModalProps });
    },
    [getModalProps, openOrUpdate],
  );

  /**
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal.
   */
  useEffect(() => {
    if (modalRef.current) openOrUpdate(getModalProps());
  }, [getModalProps, modalRef, openOrUpdate, workspaceNameValue]);

  return { modalOpen, modalRef, ...modalHook };
};

export default useModalWorkspaceDelete;
