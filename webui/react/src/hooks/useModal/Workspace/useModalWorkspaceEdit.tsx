import { Input } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { ChangeEvent, useCallback, useEffect, useMemo, useState } from 'react';

import useModal, { ModalHooks } from 'hooks/useModal/useModal';
import { patchWorkspace } from 'services/api';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { Workspace } from 'types';
import handleError from 'utils/error';

import css from './useModalWorkspaceEdit.module.scss';

interface Props {
  onClose?: () => void;
  workspace: Workspace;
}

const useModalWorkspaceEdit = ({ onClose, workspace }: Props): ModalHooks => {
  const [ name, setName ] = useState(workspace.name);

  const handleClose = useCallback(() => {
    onClose?.();
  }, [ onClose ]);

  const { modalClose, modalOpen: openOrUpdate, modalRef } = useModal({ onClose: handleClose });

  const handleNameInput = useCallback((e: ChangeEvent<HTMLInputElement>) => {
    setName(e.target.value);
  }, []);

  const modalContent = useMemo(() => {
    return (
      <div className={css.base}>
        <label className={css.label} htmlFor="name">Name</label>
        <Input id="name" value={name} onChange={handleNameInput} />
      </div>
    );
  }, [ handleNameInput, name ]);

  const handleOk = useCallback(async () => {
    try {
      await patchWorkspace({ id: workspace.id, name: name });
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to edit workspace.',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [ name, workspace.id ]);

  const getModalProps = useCallback((name: string): ModalFuncProps => {
    return {
      closable: true,
      content: modalContent,
      icon: null,
      okButtonProps: { disabled: name.length === 0 },
      okText: 'Save changes',
      onOk: handleOk,
      title: 'Edit Workspace',
    };
  }, [ handleOk, modalContent ]);

  const modalOpen = useCallback((initialModalProps: ModalFuncProps = {}) => {
    setName(workspace.name);
    openOrUpdate({ ...getModalProps(workspace.name), ...initialModalProps });
  }, [ getModalProps, openOrUpdate, workspace.name ]);

  /*
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal
   */
  useEffect(() => {
    if (modalRef.current) openOrUpdate(getModalProps(name));
  }, [ getModalProps, modalRef, name, openOrUpdate ]);

  return { modalClose, modalOpen, modalRef };
};

export default useModalWorkspaceEdit;
