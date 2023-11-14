import { Modal } from 'hew/Modal';
import { useToast } from 'hew/Toast';
import React from 'react';

import { deleteGroup } from 'services/api';
import { V1GroupSearchResult } from 'services/api-ts-sdk';
import handleError, { ErrorType } from 'utils/error';

export const API_SUCCESS_MESSAGE = 'Group deleted.';
export const MODAL_HEADER = 'Delete Group';

interface Props {
  group: V1GroupSearchResult;
  onClose?: () => void;
}

const DeleteGroupModalComponent: React.FC<Props> = ({ onClose, group }: Props) => {
  const { openToast } = useToast();

  const handleSubmit = async () => {
    if (!group.group.groupId) return;
    try {
      await deleteGroup({ groupId: group.group.groupId });
      openToast({ severity: 'Confirm', title: API_SUCCESS_MESSAGE });
      onClose?.();
    } catch (e) {
      openToast({ severity: 'Error', title: 'error deleting group' });
      handleError(e, { silent: true, type: ErrorType.Input });

      // Re-throw error to prevent modal from getting dismissed.
      throw e;
    }
  };

  return (
    <Modal
      cancel
      danger
      size="small"
      submit={{
        handleError,
        handler: handleSubmit,
        text: 'Delete',
      }}
      title={MODAL_HEADER}>
      Are you sure you want to delete group {group.group?.name} (ID: {group.group?.groupId}).
    </Modal>
  );
};

export default DeleteGroupModalComponent;
