import { useStore } from 'contexts/Store';
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

interface PermissionsHook {
  canAssignRoles: (arg0: WorkspacePermissionsArgs) => boolean;
  canCreateWorkspace: boolean;
  canCreateExperiment: (arg0: WorkspacePermissionsArgs) => boolean;
  canDeleteExperiment: (arg0: ExperimentPermissionsArgs) => boolean;
  canDeleteModel: (arg0: ModelPermissionsArgs) => boolean;
  canDeleteModelVersion: (arg0: ModelVersionPermissionsArgs) => boolean;
  canDeleteProjects: (arg0: ProjectPermissionsArgs) => boolean;
  canDeleteWorkspace: (arg0: WorkspacePermissionsArgs) => boolean;
  canGetPermissions: boolean;
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

  // Determine if the user has access to any workspaces
  // Should be updated to check user assignments and roles once available
  const canViewWorkspaces =
    relevantPermissions(userAssignments, userRoles).has('oss_user') ||
    relevantPermissions(userAssignments, userRoles).has('view_workspaces');

  return {
    canAssignRoles: (args: WorkspacePermissionsArgs) =>
      canAssignRoles(args.workspace, user, userAssignments, userRoles),
    canCreateWorkspace: canCreateWorkspace(userAssignments, userRoles),
    canCreateExperiment: (args: WorkspacePermissionsArgs) =>
      canCreateExperiment(args.workspace, userAssignments, userRoles),
    canDeleteExperiment: (args: ExperimentPermissionsArgs) =>
      canDeleteExperiment(args.experiment, user, userAssignments, userRoles),
    canDeleteModel: (args: ModelPermissionsArgs) =>
      canDeleteModel(args.model, user, userAssignments, userRoles),
    canDeleteModelVersion: (args: ModelVersionPermissionsArgs) =>
      canDeleteModelVersion(args.modelVersion, user, userAssignments, userRoles),
    canDeleteProjects: (args: ProjectPermissionsArgs) =>
      canDeleteWorkspaceProjects(args.workspace, args.project, user, userAssignments, userRoles),
    canDeleteWorkspace: (args: WorkspacePermissionsArgs) =>
      canDeleteWorkspace(args.workspace, user, userAssignments, userRoles),
    canGetPermissions: canAdministrateUsers(user, userAssignments, userRoles),
    canModifyGroups: canModifyGroups(user, userAssignments, userRoles),
    canModifyPermissions: canAdministrateUsers(user, userAssignments, userRoles),
    canModifyProjects: (args: ProjectPermissionsArgs) =>
      canModifyWorkspaceProjects(args.workspace, args.project, user, userAssignments, userRoles),
    canModifyUsers: canAdministrateUsers(user, userAssignments, userRoles),
    canModifyWorkspace: (args: WorkspacePermissionsArgs) =>
      canModifyWorkspace(args.workspace, user, userAssignments, userRoles),
    canMoveExperiment: (args: ExperimentPermissionsArgs) =>
      canMoveExperiment(args.experiment, user, userAssignments, userRoles),
    canMoveProjects: (args: ProjectPermissionsArgs) =>
      canMoveWorkspaceProjects(args.workspace, args.project, user, userAssignments, userRoles),
    canUpdateRoles: (args: WorkspacePermissionsArgs) =>
      canUpdateRoles(args.workspace, user, userAssignments, userRoles),
    canViewExperimentArtifacts: (args: WorkspacePermissionsArgs) =>
      canViewExperimentArtifacts(args.workspace, userAssignments, userRoles),
    canViewGroups: canViewGroups(user, userAssignments, userRoles),
    canViewUsers: canAdministrateUsers(user, userAssignments, userRoles),
    canViewWorkspace: (args: WorkspacePermissionsArgs) =>
      canViewWorkspace(args.workspace, userAssignments, userRoles),
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
const canAdministrateUsers = (
  user?: DetailedUser,
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[],
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles);
  return (
    !!user &&
    (permitted.has('oss_user') ? user.isAdmin : permitted.has('PERMISSION_CAN_ADMINISTRATE_USERS'))
  );
};

const canViewGroups = (
  user?: DetailedUser,
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[],
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles);
  return !!user && (permitted.has('oss_user') ? user.isAdmin : true);
};

const canModifyGroups = (
  user?: DetailedUser,
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[],
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles);
  return (
    !!user &&
    (permitted.has('oss_user') ? user.isAdmin : permitted.has('PERMISSION_CAN_UPDATE_GROUP'))
  );
};

// Experiment actions
const canCreateExperiment = (
  workspace?: PermissionWorkspace,
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[],
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return !!workspace && (permitted.has('oss_user') || permitted.has('create_experiment'));
};

const canDeleteExperiment = (
  experiment: ProjectExperiment,
  user?: DetailedUser,
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[],
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, experiment.workspaceId);
  return (
    !!experiment &&
    !!user &&
    (permitted.has('oss_user')
      ? user.isAdmin || user.id === experiment.userId
      : permitted.has('delete_experiment'))
  );
};

const canMoveExperiment = (
  experiment: ProjectExperiment,
  user?: DetailedUser,
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[],
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, experiment.workspaceId);
  return (
    !!experiment &&
    !!user &&
    (permitted.has('oss_user')
      ? user.isAdmin || user.id === experiment.userId
      : permitted.has('move_experiment'))
  );
};

// experiment artifacts (usually checkpoints)
const canViewExperimentArtifacts = (
  workspace?: PermissionWorkspace,
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[],
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return !!workspace && (permitted.has('oss_user') || permitted.has('view_experiment_artifacts'));
};

// User actions
const canGetPermissions = (
  user?: DetailedUser,
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[],
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles);
  return !!user && (permitted.has('oss_user') ? user.isAdmin : permitted.has('view_permissions'));
};

// Model and ModelVersion actions
const canDeleteModel = (
  model: ModelItem,
  user?: DetailedUser,
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[],
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles);
  return (
    !!model &&
    !!user &&
    (permitted.has('oss_user')
      ? user.isAdmin || user.id === model.userId
      : permitted.has('delete_model'))
  );
};

const canDeleteModelVersion = (
  modelVersion?: ModelVersion,
  user?: DetailedUser,
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[],
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles);
  return (
    !!modelVersion &&
    !!user &&
    (permitted.has('oss_user')
      ? user.isAdmin || user.id === modelVersion.userId
      : permitted.has('delete_model_version'))
  );
};

// Project actions
// Currently the smallest scope is workspace
const canDeleteWorkspaceProjects = (
  workspace?: PermissionWorkspace,
  project?: Project,
  user?: DetailedUser,
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[],
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    !!workspace &&
    !!user &&
    !!project &&
    (permitted.has('oss_user')
      ? user.isAdmin || user.id === project.userId
      : permitted.has('delete_projects'))
  );
};

const canModifyWorkspaceProjects = (
  workspace?: PermissionWorkspace,
  project?: Project,
  user?: DetailedUser,
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[],
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    !!workspace &&
    !!user &&
    !!project &&
    (permitted.has('oss_user')
      ? user.isAdmin || user.id === project.userId
      : permitted.has('modify_projects'))
  );
};

const canMoveWorkspaceProjects = (
  workspace?: PermissionWorkspace,
  project?: Project,
  user?: DetailedUser,
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[],
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    !!workspace &&
    !!user &&
    !!project &&
    (permitted.has('oss_user')
      ? user.isAdmin || user.id === project.userId
      : permitted.has('move_projects'))
  );
};

// Workspace actions
const canCreateWorkspace = (
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[],
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles);
  return permitted.has('oss_user') || permitted.has('create_workspace');
};

const canDeleteWorkspace = (
  workspace?: PermissionWorkspace,
  user?: DetailedUser,
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[],
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    !!workspace &&
    !!user &&
    (permitted.has('oss_user')
      ? user.isAdmin || user.id === workspace.userId
      : permitted.has('delete_workspace'))
  );
};

const canModifyWorkspace = (
  workspace?: PermissionWorkspace,
  user?: DetailedUser,
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[],
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    !!workspace &&
    !!user &&
    (permitted.has('oss_user')
      ? user.isAdmin || user.id === workspace.userId
      : permitted.has('modify_workspace'))
  );
};

const canViewWorkspace = (
  workspace?: PermissionWorkspace,
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[],
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return !!workspace && (permitted.has('oss_user') || permitted.has('view_workspace'));
};

const canUpdateRoles = (
  workspace?: PermissionWorkspace,
  user?: DetailedUser,
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[],
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    !!workspace &&
    !!user &&
    (permitted.has('oss_user')
      ? user.isAdmin || user.id === workspace.userId
      : permitted.has('update_roles'))
  );
};

const canAssignRoles = (
  workspace?: PermissionWorkspace,
  user?: DetailedUser,
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[],
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    !!workspace &&
    !!user &&
    (permitted.has('oss_user')
      ? user.isAdmin || user.id === workspace.userId
      : permitted.has('assign_roles'))
  );
};

export default usePermissions;
