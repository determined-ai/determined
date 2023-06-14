import React from 'react';

import { Modal } from 'components/kit/Modal';
import { deleteGroup } from 'services/api';
import { V1GroupSearchResult } from 'services/api-ts-sdk';
import { message } from 'utils/dialogApi';
import { ErrorType } from 'utils/error';
import handleError from 'utils/error';

export const API_SUCCESS_MESSAGE = 'Group deleted.';
export const MODAL_HEADER = 'Delete Group';

interface Props {
  group: V1GroupSearchResult;
  onClose?: () => void;
}

const DeleteGroupModalComponent: React.FC<Props> = ({ onClose, group }: Props) => {
  const handleSubmit = async () => {
    if (!group.group.groupId) return;
    try {
      await deleteGroup({ groupId: group.group.groupId });
      message.success(API_SUCCESS_MESSAGE);
      onClose?.();
    } catch (e) {
      message.error('error deleting group');
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
