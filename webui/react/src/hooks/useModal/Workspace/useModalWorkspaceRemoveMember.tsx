import { ModalFuncProps } from 'antd/es/modal/Modal';
import React, { useCallback, useEffect, useMemo } from 'react';

import useModal, { ModalHooks } from 'shared/hooks/useModal/useModal';
import { UserOrGroup } from 'types';
import { DetError, ErrorLevel, ErrorType } from 'shared/utils/error';
import handleError from 'utils/error';
import { createAssignmentRequest } from 'utils/user';
import { removeAssignments } from 'services/api';
import css from './useModalWorkspaceRemoveMember.module.scss';

interface Props {
  userOrGroupId: number;
  userOrGroup: UserOrGroup;
  name: string;
  onClose?: () => void;
  workspaceId: number;
}

const useModalWorkspaceRemoveMember = ({
  onClose,
  userOrGroup,
  workspaceId,
  name,
  userOrGroupId,
}: Props): ModalHooks => {
  const { modalOpen: openOrUpdate, modalRef, ...modalHook } = useModal({ onClose });

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
      await removeAssignments(createAssignmentRequest(userOrGroup, userOrGroupId, 0, workspaceId));
    } catch (e) {
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
    return;
  }, [userOrGroup]);

  const getModalProps = useCallback((): ModalFuncProps => {
    return {
      closable: true,
      content: modalContent,
      icon: null,
      okButtonProps: { danger: true },
      okText: 'Remove',
      onOk: handleOk,
      title: `Remove ${name}`,
    };
  }, [modalContent, name]);

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
  }, [getModalProps, modalRef, name, openOrUpdate]);

  return { modalOpen, modalRef, ...modalHook };
};

export default useModalWorkspaceRemoveMember;
