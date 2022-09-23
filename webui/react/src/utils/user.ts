import {
  V1Group,
  V1RoleAssignment,
  V1AssignRolesRequest,
  V1RemoveAssignmentsRequest,
} from 'services/api-ts-sdk';
import { DetailedUser, UserOrGroup, User } from 'types';

interface UserNameFields {
  displayName?: string;
  username?: string;
}

export function getDisplayName(user: DetailedUser | User | UserNameFields | undefined): string {
  return user?.displayName || user?.username || 'Unavailable';
}

export function isUser(obj: UserOrGroup): string | undefined {
  const user = obj as User;
  return user?.username || user?.displayName;
}

export function getName(obj: UserOrGroup): string {
  const user = obj as User;
  const group = obj as V1Group;
  return isUser(obj) ? getDisplayName(user) : group.name ? group.name : '';
}

export const getIdFromUserOrGroup = (obj: UserOrGroup): number => {
  if (isUser(obj)) {
    const user = obj as User;
    return user.id;
  }
  const group = obj as V1Group;

  // THe groupId should always exist
  return group.groupId || 0;
};

export function createAssignmentRequest(
  userOrGroup: UserOrGroup,
  userOrGroupId: number,
  roleId: number,
  workspaceId: number,
): V1AssignRolesRequest | V1RemoveAssignmentsRequest {
  const roleAssignment: V1RoleAssignment = {
    role: {
      roleId: 0,
    },
    scopeWorkspaceId: workspaceId,
  };
  const assignment = isUser(userOrGroup)
    ? {
        userRoleAssignments: [
          {
            userId: userOrGroupId,
            roleAssignment,
          },
        ],
      }
    : {
        groupRoleAssignments: [
          {
            groupId: userOrGroupId,
            roleAssignment,
          },
        ],
      };
  return assignment;
}
