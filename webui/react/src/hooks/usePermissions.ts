import { useStore } from 'contexts/Store';
import useFeature from 'hooks/useFeature';
import {
  DetailedUser,
  ExperimentPermissionsArgs,
  ModelItem,
  ModelVersion,
  Permission,
  PermissionWorkspace,
  Project,
  ProjectExperiment,
  UserAssignment,
  UserRole,
  WorkspacePermissionsArgs,
} from 'types';

interface ModelPermissionsArgs {
  model: ModelItem;
}

interface ModelVersionPermissionsArgs {
  modelVersion?: ModelVersion;
}

interface ProjectPermissionsArgs {
  project?: Project;
  workspace?: PermissionWorkspace;
}

interface RbacOptsProps {
  rbacEnabled: boolean;
  user?: DetailedUser;
  userAssignments?: UserAssignment[];
  userRoles?: UserRole[];
}

interface PermissionsHook {
  canAssignRoles: (arg0: WorkspacePermissionsArgs) => boolean;
  canCreateExperiment: (arg0: WorkspacePermissionsArgs) => boolean;
  canCreateProject: (arg0: WorkspacePermissionsArgs) => boolean;
  canCreateWorkspace: boolean;
  canDeleteExperiment: (arg0: ExperimentPermissionsArgs) => boolean;
  canDeleteModel: (arg0: ModelPermissionsArgs) => boolean;
  canDeleteModelVersion: (arg0: ModelVersionPermissionsArgs) => boolean;
  canDeleteProjects: (arg0: ProjectPermissionsArgs) => boolean;
  canDeleteWorkspace: (arg0: WorkspacePermissionsArgs) => boolean;
  canGetPermissions: boolean;
  canModifyExperiment: (arg0: WorkspacePermissionsArgs) => boolean;
  canModifyExperimentMetadata: (arg0: WorkspacePermissionsArgs) => boolean;
  canModifyGroups: boolean;
  canModifyPermissions: boolean;
  canModifyProjects: (arg0: ProjectPermissionsArgs) => boolean;
  canModifyUsers: boolean;
  canModifyWorkspace: (arg0: WorkspacePermissionsArgs) => boolean;
  canMoveExperiment: (arg0: ExperimentPermissionsArgs) => boolean;
  canMoveProjects: (arg0: ProjectPermissionsArgs) => boolean;
  canUpdateRoles: (arg0: ProjectPermissionsArgs) => boolean;
  canViewExperimentArtifacts: (arg0: WorkspacePermissionsArgs) => boolean;
  canViewGroups: boolean;
  canViewUsers: boolean;
  canViewWorkspace: (arg0: WorkspacePermissionsArgs) => boolean;
  canViewWorkspaces: boolean;
}

const usePermissions = (): PermissionsHook => {
  const {
    auth: { user },
    userAssignments,
    userRoles,
  } = useStore();
  const rbacOpts = {
    rbacEnabled: useFeature().isOn('rbac'),
    user,
    userAssignments,
    userRoles,
  };

  // Determine if the user has access to any workspaces
  // Should be updated to check user assignments and roles once available
  const canViewWorkspaces =
    relevantPermissions(userAssignments, userRoles).has('oss_user') ||
    relevantPermissions(userAssignments, userRoles).has('view_workspaces');

  return {
    canAssignRoles: (args: WorkspacePermissionsArgs) => canAssignRoles(rbacOpts, args.workspace),
    canCreateExperiment: (args: WorkspacePermissionsArgs) =>
      canCreateExperiment(rbacOpts, args.workspace),
    canCreateProject: (args: WorkspacePermissionsArgs) =>
      canCreateProject(rbacOpts, args.workspace),
    canCreateWorkspace: canCreateWorkspace(rbacOpts),
    canDeleteExperiment: (args: ExperimentPermissionsArgs) =>
      canDeleteExperiment(rbacOpts, args.experiment),
    canDeleteModel: (args: ModelPermissionsArgs) => canDeleteModel(rbacOpts, args.model),
    canDeleteModelVersion: (args: ModelVersionPermissionsArgs) =>
      canDeleteModelVersion(rbacOpts, args.modelVersion),
    canDeleteProjects: (args: ProjectPermissionsArgs) =>
      canDeleteWorkspaceProjects(rbacOpts, args.workspace, args.project),
    canDeleteWorkspace: (args: WorkspacePermissionsArgs) =>
      canDeleteWorkspace(rbacOpts, args.workspace),
    canGetPermissions: canGetPermissions(rbacOpts),
    canModifyExperiment: (args: WorkspacePermissionsArgs) =>
      canModifyExperiment(rbacOpts, args.workspace),
    canModifyExperimentMetadata: (args: WorkspacePermissionsArgs) =>
      canModifyExperimentMetadata(rbacOpts, args.workspace),
    canModifyGroups: canModifyGroups(rbacOpts),
    canModifyPermissions: canAdministrateUsers(rbacOpts),
    canModifyProjects: (args: ProjectPermissionsArgs) =>
      canModifyWorkspaceProjects(rbacOpts, args.workspace, args.project),
    canModifyUsers: canAdministrateUsers(rbacOpts),
    canModifyWorkspace: (args: WorkspacePermissionsArgs) =>
      canModifyWorkspace(rbacOpts, args.workspace),
    canMoveExperiment: (args: ExperimentPermissionsArgs) =>
      canMoveExperiment(rbacOpts, args.experiment),
    canMoveProjects: (args: ProjectPermissionsArgs) =>
      canMoveWorkspaceProjects(rbacOpts, args.workspace, args.project),
    canUpdateRoles: (args: WorkspacePermissionsArgs) => canUpdateRoles(rbacOpts, args.workspace),
    canViewExperimentArtifacts: (args: WorkspacePermissionsArgs) =>
      canViewExperimentArtifacts(rbacOpts, args.workspace),
    canViewGroups: canViewGroups(rbacOpts),
    canViewUsers: canAdministrateUsers(rbacOpts),
    canViewWorkspace: (args: WorkspacePermissionsArgs) =>
      canViewWorkspace(rbacOpts, args.workspace),
    canViewWorkspaces,
  };
};

// Permissions inside this workspace scope (no workspace = cluster-wide scope)
// Typically returns a Set<string> of permissions.
const relevantPermissions = (
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[],
  workspaceId?: number,
): { has: (arg0: string) => boolean } => {
  if (!userAssignments || !userRoles) {
    // console.error('missing UserAssignment or UserRole');
    return new Set<string>();
  }
  const relevantAssigned = userAssignments
    .filter((a) => a.cluster || (workspaceId && a.workspaces && a.workspaces.includes(workspaceId)))
    .map((a) => a.name);
  let permissions = Array<Permission>();
  userRoles
    .filter((r) => relevantAssigned.includes(r.name))
    .forEach((r) => {
      // TODO: is it possible a role is assigned to this workspace,
      // but not all of its permissions?
      permissions = permissions.concat(r.permissions.filter((p) => p.isGlobal || workspaceId));
    });
  const permitter = new Set<string>(permissions.map((p) => p.name));
  // a cluster_admin has all permissions
  if (permitter.has('cluster_admin')) {
    return { has: () => true };
  }
  return permitter;
};

// User actions
const canAdministrateUsers = ({
  rbacEnabled,
  user,
  userAssignments,
  userRoles,
}: RbacOptsProps): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles);
  return (
    !!user && (rbacEnabled ? permitted.has('PERMISSION_CAN_ADMINISTRATE_USERS') : user.isAdmin)
  );
};

const canViewGroups = ({ rbacEnabled, user }: RbacOptsProps): boolean => {
  return !!user && (rbacEnabled || user.isAdmin);
};

const canModifyGroups = ({
  rbacEnabled,
  user,
  userAssignments,
  userRoles,
}: RbacOptsProps): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles);
  return !!user && (rbacEnabled ? permitted.has('PERMISSION_CAN_UPDATE_GROUP') : user.isAdmin);
};

// Experiment actions
const canCreateExperiment = (
  { rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return !!workspace && (!rbacEnabled || permitted.has('create_experiment'));
};

const canDeleteExperiment = (
  { rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  experiment: ProjectExperiment,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, experiment.workspaceId);
  return (
    !!experiment &&
    !!user &&
    (rbacEnabled
      ? permitted.has('delete_experiment')
      : user.isAdmin || user.id === experiment.userId)
  );
};

const canModifyExperiment = (
  { rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return !!workspace && (!rbacEnabled || permitted.has('update_experiments'));
};

const canModifyExperimentMetadata = (
  { rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return !!workspace && (!rbacEnabled || permitted.has('update_experiment_metadata'));
};

const canMoveExperiment = (
  { rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  experiment: ProjectExperiment,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, experiment.workspaceId);
  return (
    !!experiment &&
    !!user &&
    (rbacEnabled ? permitted.has('move_experiment') : user.isAdmin || user.id === experiment.userId)
  );
};

// experiment artifacts (checkpoints, metrics, etc.)
const canViewExperimentArtifacts = (
  { rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return !!workspace && (!rbacEnabled || permitted.has('view_experiment_artifacts'));
};

// User actions
const canGetPermissions = ({
  rbacEnabled,
  user,
  userAssignments,
  userRoles,
}: RbacOptsProps): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles);
  return !!user && (rbacEnabled ? permitted.has('view_permissions') : user.isAdmin);
};

// Model and ModelVersion actions
const canDeleteModel = (
  { rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  model: ModelItem,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles);
  return (
    !!model &&
    !!user &&
    (rbacEnabled ? permitted.has('delete_model') : user.isAdmin || user.id === model.userId)
  );
};

const canDeleteModelVersion = (
  { rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  modelVersion?: ModelVersion,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles);
  return (
    !!modelVersion &&
    !!user &&
    (rbacEnabled
      ? permitted.has('delete_model_version')
      : user.isAdmin || user.id === modelVersion.userId)
  );
};

// Project actions
// Currently the smallest scope is workspace
const canCreateProject = (
  { rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return !rbacEnabled || permitted.has('create_project');
};

const canDeleteWorkspaceProjects = (
  { rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
  project?: Project,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    !!workspace &&
    !!user &&
    !!project &&
    (rbacEnabled ? permitted.has('delete_projects') : user.isAdmin || user.id === project.userId)
  );
};

const canModifyWorkspaceProjects = (
  { rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
  project?: Project,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    !!workspace &&
    !!user &&
    !!project &&
    (rbacEnabled ? permitted.has('modify_projects') : user.isAdmin || user.id === project.userId)
  );
};

const canMoveWorkspaceProjects = (
  { rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
  project?: Project,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    !!workspace &&
    !!user &&
    !!project &&
    (rbacEnabled ? permitted.has('move_projects') : user.isAdmin || user.id === project.userId)
  );
};

// Workspace actions
const canCreateWorkspace = ({
  rbacEnabled,
  userAssignments,
  userRoles,
}: RbacOptsProps): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles);
  return !rbacEnabled || permitted.has('create_workspace');
};

const canDeleteWorkspace = (
  { rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    !!workspace &&
    !!user &&
    (rbacEnabled ? permitted.has('delete_workspace') : user.isAdmin || user.id === workspace.userId)
  );
};

const canModifyWorkspace = (
  { rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    !!workspace &&
    !!user &&
    (rbacEnabled ? permitted.has('modify_workspace') : user.isAdmin || user.id === workspace.userId)
  );
};

const canViewWorkspace = (
  { rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return !!workspace && (!rbacEnabled || permitted.has('view_workspace'));
};

const canUpdateRoles = (
  { rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    !!workspace &&
    !!user &&
    (rbacEnabled ? permitted.has('update_roles') : user.isAdmin || user.id === workspace.userId)
  );
};

const canAssignRoles = (
  { rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    !!workspace &&
    !!user &&
    (rbacEnabled ? permitted.has('assign_roles') : user.isAdmin || user.id === workspace.userId)
  );
};

export default usePermissions;
