import { Select } from 'antd';
import { RawValueType } from 'rc-select/lib/BaseSelect';
import { LabelInValueType } from 'rc-select/lib/Select';
import React, { useState } from 'react';

import useFeature from 'hooks/useFeature';
import {
  assignRolesToGroup,
  assignRolesToUser,
  removeRolesFromGroup,
  removeRolesFromUser,
} from 'services/api';
import { V1Role, V1RoleWithAssignments } from 'services/api-ts-sdk/models';
import { UserOrGroup } from 'types';
import handleError from 'utils/error';
import { getAssignedRole, getIdFromUserOrGroup, isUser } from 'utils/user';

import css from './RoleRenderer.module.scss';

interface Props {
  assignments: V1RoleWithAssignments[];
  rolesAssignableToScope: V1Role[];
  userCanAssignRoles: boolean;
  userOrGroup: UserOrGroup;
  workspaceId: number;
}

const RoleRenderer: React.FC<Props> = ({
  assignments,
  rolesAssignableToScope,
  userOrGroup,
  userCanAssignRoles,
  workspaceId,
}) => {
  const roleAssignment = getAssignedRole(userOrGroup, assignments);
  const [memberRoleId, setMemberRole] = useState(roleAssignment?.role?.roleId);
  let knownRoles = rolesAssignableToScope;

  const mockWorkspaceMembers = useFeature().isOn('mock_workspace_members');

  knownRoles = mockWorkspaceMembers
    ? [
        {
          name: 'Editor',
          permissions: [],
          roleId: 1,
        },
        {
          name: 'Viewer',
          permissions: [],
          roleId: 2,
        },
      ]
    : knownRoles;

  return (
    <Select
      className={css.base}
      disabled={!userCanAssignRoles || !roleAssignment?.scopeWorkspaceId}
      value={memberRoleId}
      onSelect={async (value: RawValueType | LabelInValueType) => {
        const roleIdValue = value as number;
        const userOrGroupId = getIdFromUserOrGroup(userOrGroup);
        const oldRoleIds = memberRoleId ? [memberRoleId] : [];
        try {
          // Try to remove the old role and then add the new role
          isUser(userOrGroup)
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
            isUser(userOrGroup)
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
      {knownRoles.map((role) => (
        <Select.Option key={role.roleId} value={role.roleId}>
          {role.name}
        </Select.Option>
      ))}
    </Select>
  );
};

export default RoleRenderer;
