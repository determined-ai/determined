import { useStore } from 'contexts/Store';
import { Project } from 'types';
import {
  canDeleteWorkspaceProjects,
  canModifyWorkspace,
  canModifyWorkspaceProjects,
  canMoveWorkspaceProjects,
  PermissionWorkspace,
} from 'utils/role';

interface PermissionsConfig {
  project?: Project;
  workspace?: PermissionWorkspace;
}

type PermissionFn = (arg0: PermissionsConfig) => boolean;

interface PermissionsHook {
  canDeleteProjects: PermissionFn;
  canModifyProjects: PermissionFn;
  canModifyWorkspace: PermissionFn;
  canMoveProjects: PermissionFn;
}

const usePermissions = (): PermissionsHook => {
  const { auth: { user }, userAssignments, userRoles } = useStore();

  return {
    canDeleteProjects: (config: PermissionsConfig) => canDeleteWorkspaceProjects(
      config.workspace,
      config.project,
      user,
      userAssignments,
      userRoles,
    ),
    canModifyProjects: (config: PermissionsConfig) => canModifyWorkspaceProjects(
      config.workspace,
      config.project,
      user,
      userAssignments,
      userRoles,
    ),
    canModifyWorkspace: (config: PermissionsConfig) => canModifyWorkspace(
      config.workspace,
      user,
      userAssignments,
      userRoles,
    ),
    canMoveProjects: (config: PermissionsConfig) => canMoveWorkspaceProjects(
      config.workspace,
      config.project,
      user,
      userAssignments,
      userRoles,
    ),
  };
};

export default usePermissions;
