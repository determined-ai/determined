import { message } from 'antd';
import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import { removeRolesFromGroup, removeRolesFromUser } from 'services/api';
import useModal, { ModalHooks } from 'shared/hooks/useModal/useModal';
import { DetError, ErrorLevel, ErrorType } from 'shared/utils/error';
import { UserOrGroup } from 'types';
import handleError from 'utils/error';
import { isUser } from 'utils/user';

import css from './useModalWorkspaceRemoveMember.module.scss';

interface Props {
  name: string;
  onClose?: () => void;
  roleIds: number[];
  scopeWorkspaceId: number;
  userOrGroup: UserOrGroup;
  userOrGroupId: number;
}

const useModalWorkspaceRemoveMember = ({
  onClose,
  userOrGroup,
  name,
  roleIds,
  scopeWorkspaceId,
  userOrGroupId,
}: Props): ModalHooks => {
  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal();
  const [isDeleting, setIsDeleting] = useState(false);

  const modalContent = useMemo(() => {
    return (
      <div className={css.base}>
        <p>
          Are you sure you want to remove {name} from this workspace? They will no longer be able to
          access the contents of this workspace. Nothing will be deleted.
        </p>
      </div>
    );
  }, [name]);

  const handleOk = useCallback(async () => {
    try {
      setIsDeleting(true);
      isUser(userOrGroup)
        ? await removeRolesFromUser({ roleIds, scopeWorkspaceId, userId: userOrGroupId })
        : await removeRolesFromGroup({ groupId: userOrGroupId, roleIds, scopeWorkspaceId });
      onClose?.();
      message.success(`${name} removed from workspace`);
    } catch (e) {
      setIsDeleting(false);
      if (e instanceof DetError) {
        handleError(e, {
          level: e.level,
          publicMessage: e.publicMessage,
          publicSubject: 'Unable to remove user or group from workspace.',
          silent: false,
          type: e.type,
        });
      } else {
        handleError(e, {
          level: ErrorLevel.Error,
          publicMessage: 'Please try again later.',
          publicSubject: 'Unable to remove user or group.',
          silent: false,
          type: ErrorType.Server,
        });
      }
    }
  }, [name, roleIds, scopeWorkspaceId, userOrGroup, userOrGroupId, onClose]);

  const getModalProps = useCallback((): ModalFuncProps => {
    return {
      closable: true,
      content: modalContent,
      icon: null,
      okButtonProps: { danger: true, disabled: isDeleting },
      okText: 'Remove',
      onOk: handleOk,
      title: `Remove ${name}`,
    };
  }, [handleOk, modalContent, name, isDeleting]);

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
    if (modalRef.current && !isDeleting) openOrUpdate(getModalProps());
  }, [getModalProps, isDeleting, modalRef, name, openOrUpdate]);

  return { modalOpen, modalRef, ...modalHook };
};

export default useModalWorkspaceRemoveMember;
