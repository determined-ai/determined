import { Select } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { useCallback, useEffect, useMemo } from 'react';

import { useStore } from 'contexts/Store';
import useModal, { ModalHooks } from 'shared/hooks/useModal/useModal';
import { Workspace } from 'types';
import { getDisplayName } from 'utils/user';

import css from './useModalWorkspaceAddMember.module.scss';

interface Props {
  onClose?: () => void;
  workspace: Workspace;
}

// Adding this lint rule to keep the reference to the workspace
// which will be needed when calling the API.
/* eslint-disable-next-line @typescript-eslint/no-unused-vars */
const useModalWorkspaceAddMember = ({ onClose, workspace }: Props): ModalHooks => {
  const { users } = useStore();
  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal({ onClose });

  const modalContent = useMemo(() => {
    return (
      <div className={css.base}>
        <Select
          placeholder="Find user or group by display name or username"
          showSearch>
          {users.map((u) => (
            <Select.Option key={u.id} value={u.id}>
              {getDisplayName(u)}
            </Select.Option>
            ))}
        </Select>
      </div>
    );
  }, [ users ]);

  const getModalProps = useCallback((): ModalFuncProps => {
    return {
      closable: true,
      content: modalContent,
      icon: null,
      okButtonProps: { danger: true },
      okText: 'Add Member',
      title: 'Add Member',
    };
  }, [ modalContent ]);

  const modalOpen = useCallback((initialModalProps: ModalFuncProps = {}) => {
    openOrUpdate({ ...getModalProps(), ...initialModalProps });
  }, [ getModalProps, openOrUpdate ]);

  /**
   * When modal props changes are detected, such as modal content
   * title, and buttons, update the modal.
   */
  useEffect(() => {
    if (modalRef.current) openOrUpdate(getModalProps());
  }, [ getModalProps, modalRef, openOrUpdate ]);

  return { modalOpen, modalRef, ...modalHook };
};

export default useModalWorkspaceAddMember ;
