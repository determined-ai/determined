import { Input } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { ChangeEvent, useCallback, useEffect, useMemo, useState } from 'react';

import { paths } from 'routes/utils';
import useModal, { ModalHooks } from 'shared/hooks/useModal/useModal';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { routeToReactUrl } from 'shared/utils/routes';
import { useDeleteWorkspace } from 'stores/workspaces';
import { Workspace } from 'types';
import handleError from 'utils/error';

import css from './useModalWorkspaceDelete.module.scss';

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
  const [name, setName] = useState('');

  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal({ onClose });
  const deleteWorkspace = useDeleteWorkspace();

  const handleNameInput = useCallback((e: ChangeEvent<HTMLInputElement>) => {
    setName(e.target.value);
  }, []);

  const modalContent = useMemo(() => {
    return (
      <div className={css.base}>
        <p>
          Are you sure you want to delete <strong>&quot;{workspace.name}&quot;</strong>?
        </p>
        <p>
          All projects, experiments, and notes within it will also be deleted. This cannot be
          undone.
        </p>
        <label className={css.label} htmlFor="name">
          Enter workspace name to confirm deletion.
        </label>
        <Input autoComplete="off" id="name" value={name} onChange={handleNameInput} />
      </div>
    );
  }, [handleNameInput, name, workspace.name]);

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

  const getModalProps = useCallback(
    (name = ''): ModalFuncProps => {
      return {
        closable: true,
        content: modalContent,
        icon: null,
        okButtonProps: { danger: true, disabled: name !== workspace.name },
        okText: 'Delete Workspace',
        onOk: handleOk,
        title: 'Delete Workspace',
      };
    },
    [handleOk, modalContent, workspace.name],
  );

  const modalOpen = useCallback(
    (initialModalProps: ModalFuncProps = {}) => {
      setName('');
      openOrUpdate({ ...getModalProps(), ...initialModalProps });
    },
    [getModalProps, openOrUpdate],
  );

  /**
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal.
   */
  useEffect(() => {
    if (modalRef.current) openOrUpdate(getModalProps(name));
  }, [getModalProps, modalRef, name, openOrUpdate]);

  return { modalOpen, modalRef, ...modalHook };
};

export default useModalWorkspaceDelete;
