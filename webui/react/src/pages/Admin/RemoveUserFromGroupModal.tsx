import { Modal } from 'hew/Modal';
import { useToast } from 'hew/Toast';
import { useCallback } from 'react';

import { updateGroup } from 'services/api';
import { V1GroupSearchResult, V1User } from 'services/api-ts-sdk';
import handleError, { ErrorType } from 'utils/error';

interface Props {
  groupResult: V1GroupSearchResult;
  user?: V1User;
  fetchGroups: () => Promise<void>;
  onExpand: (expand: boolean, record: V1GroupSearchResult) => void;
}

const RemoveUserFromGroupModalComponent = ({
  groupResult,
  user,
  fetchGroups,
  onExpand,
}: Props): JSX.Element => {
  const { openToast } = useToast();
  const onRemoveUser = useCallback(
    async (record: V1GroupSearchResult, userId?: number) => {
      const {
        group: { groupId },
      } = record;
      if (!groupId || !userId) return;
      try {
        await updateGroup({ groupId, removeUsers: [userId] });
        openToast({
          severity: 'Confirm',
          title: 'User has been removed from group.',
        });
        onExpand(true, record);
        fetchGroups();
      } catch (e) {
        openToast({ severity: 'Error', title: 'Error deleting group.' });
        handleError(e, { silent: true, type: ErrorType.Input });
      }
    },
    [onExpand, openToast, fetchGroups],
  );

  const handleOk = useCallback(() => {
    onRemoveUser(groupResult, user?.id);
  }, [groupResult, onRemoveUser, user]);

  return (
    <Modal
      danger
      size="small"
      submit={{
        handleError,
        handler: handleOk,
        text: 'Remove User',
      }}
      title="Confirm Removing User from Group">
      <div>
        Are you sure you want to remove {user?.username ?? 'this user'} from{' '}
        {groupResult.group.name ?? 'this group'}?
      </div>
    </Modal>
  );
};

export default RemoveUserFromGroupModalComponent;
