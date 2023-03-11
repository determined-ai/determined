import { Select } from 'antd';
import { RawValueType } from 'rc-select/lib/BaseSelect';
import { LabelInValueType } from 'rc-select/lib/Select';
import React, { useState } from 'react';

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

import css from './RoleRenderer.module.scss';

interface Props {
  rolesAssignableToScope: V1Role[];
  userCanAssignRoles: boolean;
  userOrGroup: UserOrGroupWithRoleInfo;
  workspaceId: number;
}

const RoleRenderer: React.FC<Props> = ({
  rolesAssignableToScope,
  userOrGroup,
  userCanAssignRoles,
  workspaceId,
}) => {
  // const roleAssignment = getAssignedRole(userOrGroup, assignments);
  const [memberRoleId, setMemberRole] = useState(userOrGroup.roleAssignment?.role?.roleId);

  return (
    <Select
      className={css.base}
      disabled={!userCanAssignRoles || !userOrGroup.roleAssignment.scopeWorkspaceId}
      value={memberRoleId}
      onSelect={async (value: RawValueType | LabelInValueType) => {
        const roleIdValue = value as number;
        const userOrGroupId = isUserWithRoleInfo(userOrGroup)
          ? userOrGroup.userId
          : userOrGroup.groupId ?? 0;
        const oldRoleIds = memberRoleId ? [memberRoleId] : [];
        try {
          // Try to remove the old role and then add the new role
          isUserWithRoleInfo(userOrGroup)
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
          try {
            isUserWithRoleInfo(userOrGroup)
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
            setMemberRole(roleIdValue);
          } catch (addRoleError) {
            handleError(addRoleError, {
              publicSubject: 'Unable to update role for user or group unable to add new role.',
              silent: false,
            });
          }
        } catch (removeRoleError) {
          handleError(removeRoleError, {
            publicSubject:
              'Unable to update role for user or group could unable to remove current role.',
            silent: false,
          });
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
