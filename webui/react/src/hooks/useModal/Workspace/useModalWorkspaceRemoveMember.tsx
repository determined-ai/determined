import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { useCallback, useEffect, useMemo } from 'react';

import useModal, { ModalHooks } from 'shared/hooks/useModal/useModal';
import { MemberOrGroup, Workspace } from 'types';

import css from './useModalWorkspaceRemoveMember.module.scss';

interface Props {
  member: MemberOrGroup;
  name: string;
  onClose?: () => void;
  workspace: Workspace;
}

// Adding this lint rule to keep the reference to the member and workspace
// which will be needed when calling the API.
/* eslint-disable-next-line @typescript-eslint/no-unused-vars */
const useModalWorkspaceRemoveMember = ({ onClose, member, workspace, name }: Props): ModalHooks => {

  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal({ onClose });

  const modalContent = useMemo(() => {
    return (
      <div className={css.base}>
        <p>Are you sure you want to remove {name} from this workspace?
          They will no longer be able to access the contents of this workspace.
          Nothing will be deleted.
        </p>
      </div>
    );
  }, [ name ]);

  const getModalProps = useCallback((): ModalFuncProps => {
    return {
      closable: true,
      content: modalContent,
      icon: null,
      okButtonProps: { danger: true },
      okText: 'Remove',
      title: `Remove ${name}`,
    };
  }, [ modalContent, name ]);

  const modalOpen = useCallback((initialModalProps: ModalFuncProps = {}) => {
    openOrUpdate({ ...getModalProps(), ...initialModalProps });
  }, [ getModalProps, openOrUpdate ]);

  /**
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal.
   */
  useEffect(() => {
    if (modalRef.current) openOrUpdate(getModalProps());
  }, [ getModalProps, modalRef, name, openOrUpdate ]);

  return { modalOpen, modalRef, ...modalHook };
};

export default useModalWorkspaceRemoveMember ;
