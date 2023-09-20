import React, { useCallback, useState } from 'react';

import { Modal } from 'components/kit/Modal';
import { makeToast } from 'components/kit/Toast';
import { removeRolesFromGroup, removeRolesFromUser } from 'services/api';
import { UserOrGroupWithRoleInfo } from 'types';
import handleError, { DetError, ErrorLevel, ErrorType } from 'utils/error';
import { isUserWithRoleInfo } from 'utils/user';

interface Props {
  name: string;
  onClose?: () => void;
  roleIds: number[];
  scopeWorkspaceId: number;
  userOrGroup: UserOrGroupWithRoleInfo;
  userOrGroupId: number;
}

const WorkspaceMemberRemoveComponent: React.FC<Props> = ({
  onClose,
  userOrGroup,
  name,
  roleIds,
  scopeWorkspaceId,
  userOrGroupId,
}: Props) => {
  const [isDeleting, setIsDeleting] = useState<boolean>(false);

  const handleSubmit = useCallback(async () => {
    try {
      setIsDeleting(true);
      isUserWithRoleInfo(userOrGroup)
        ? await removeRolesFromUser({ roleIds, scopeWorkspaceId, userId: userOrGroupId })
        : await removeRolesFromGroup({ groupId: userOrGroupId, roleIds, scopeWorkspaceId });
      onClose?.();
      makeToast({ compact: true, severity: 'Confirm', title: `${name} removed from workspace` });
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

  return (
    <Modal
      cancel
      danger
      size="small"
      submit={{
        disabled: isDeleting,
        handleError,
        handler: handleSubmit,
        text: 'Remove',
      }}
      title={`Remove ${name}`}>
      <p>
        Are you sure you want to remove {name} from this workspace? They will no longer be able to
        access the contents of this workspace. Nothing will be deleted.
      </p>
    </Modal>
  );
};

export default WorkspaceMemberRemoveComponent;
