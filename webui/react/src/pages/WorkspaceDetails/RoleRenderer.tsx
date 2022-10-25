import { Select } from 'antd';
import { RawValueType } from 'rc-select/lib/BaseSelect';
import { LabelInValueType } from 'rc-select/lib/Select';
import React, { useState } from 'react';

import { useStore } from 'contexts/Store';
import useFeature from 'hooks/useFeature';
import {
  assignRolesToGroup,
  assignRolesToUser,
  removeRolesFromGroup,
  removeRolesFromUser,
} from 'services/api';
import { V1RoleWithAssignments } from 'services/api-ts-sdk';
import { UserOrGroup } from 'types';
import handleError from 'utils/error';
import { getAssignedRole, getIdFromUserOrGroup, isUser } from 'utils/user';

import css from './RoleRenderer.module.scss';

interface Props {
  assignments: V1RoleWithAssignments[];
  userCanAssignRoles: boolean;
  userOrGroup: UserOrGroup;
  workspaceId: number;
}

const RoleRenderer: React.FC<Props> = ({
  assignments,
  userOrGroup,
  userCanAssignRoles,
  workspaceId,
}) => {
  const roleAssignment = getAssignedRole(userOrGroup, assignments);
  const [memberRoleId, setMemberRole] = useState(roleAssignment?.role?.roleId);
  let { knownRoles } = useStore();

  const mockWorkspaceMembers = useFeature().isOn('mock_workspace_members');

  knownRoles = mockWorkspaceMembers
    ? [
        {
          id: 1,
          name: 'Editor',
          permissions: [],
        },
        {
          id: 2,
          name: 'Viewer',
          permissions: [],
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
        <Select.Option key={role.id} value={role.id}>
          {role.name}
        </Select.Option>
      ))}
    </Select>
  );
};

export default RoleRenderer;
