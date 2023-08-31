import { Select } from 'antd';
import { RawValueType } from 'rc-select/lib/BaseSelect';
import { LabelInValueType } from 'rc-select/lib/Select';
import React, { useState } from 'react';

import css from 'pages/WorkspaceDetails/RoleRenderer.module.scss';
import {
  assignRolesToGroup,
  assignRolesToUser,
  removeRolesFromGroup,
  removeRolesFromUser,
} from 'services/api';
import { V1Role } from 'services/api-ts-sdk';
import { UserOrGroupWithRoleInfo } from 'types';
import handleError from 'utils/error';
import { isUserWithRoleInfo } from 'utils/user';

interface Props {
  fetchMembers: () => void;
  rolesAssignableToScope: V1Role[];
  userCanAssignRoles: boolean;
  userOrGroupWithRoleInfo: UserOrGroupWithRoleInfo;
  workspaceId: number;
}

const RoleRenderer: React.FC<Props> = ({
  fetchMembers,
  rolesAssignableToScope,
  userOrGroupWithRoleInfo,
  userCanAssignRoles,
  workspaceId,
}) => {
  const [memberRoleId, setMemberRole] = useState(
    userOrGroupWithRoleInfo.roleAssignment?.role?.roleId,
  );

  return (
    <Select
      className={css.base}
      disabled={!userCanAssignRoles || !userOrGroupWithRoleInfo.roleAssignment.scopeWorkspaceId}
      value={memberRoleId}
      onSelect={async (value: RawValueType | LabelInValueType) => {
        const roleIdValue = value as number;
        const userOrGroupId = isUserWithRoleInfo(userOrGroupWithRoleInfo)
          ? userOrGroupWithRoleInfo.userId
          : userOrGroupWithRoleInfo.groupId ?? 0;
        const oldRoleIds = memberRoleId ? [memberRoleId] : [];
        try {
          // Try to add the new role and then remove the old role
          // to keep the permission
          isUserWithRoleInfo(userOrGroupWithRoleInfo)
            ? await assignRolesToUser({
                roleIds: [roleIdValue],
                scopeWorkspaceId: workspaceId,
                userId: userOrGroupId,
              })
            : await assignRolesToGroup({
                groupId: userOrGroupId,
                roleIds: [roleIdValue],
                scopeWorkspaceId: workspaceId,
              });

          try {
            isUserWithRoleInfo(userOrGroupWithRoleInfo)
              ? await removeRolesFromUser({
                  roleIds: oldRoleIds,
                  scopeWorkspaceId: workspaceId,
                  userId: userOrGroupId,
                })
              : await removeRolesFromGroup({
                  groupId: userOrGroupId,
                  roleIds: oldRoleIds,
                  scopeWorkspaceId: workspaceId,
                });
            setMemberRole(roleIdValue);
          } catch (addRoleError) {
            handleError(addRoleError, {
              publicSubject: 'Unable to update role for user or group unable to remove new role.',
              silent: false,
            });
          }
        } catch (removeRoleError) {
          handleError(removeRoleError, {
            publicSubject:
              'Unable to update role for user or group could unable to add current role.',
            silent: false,
          });
        } finally {
          fetchMembers();
        }
      }}>
      {rolesAssignableToScope.map((role) => (
        <Select.Option key={role.roleId} value={role.roleId}>
          {role.name}
        </Select.Option>
      ))}
    </Select>
  );
};

export default RoleRenderer;
