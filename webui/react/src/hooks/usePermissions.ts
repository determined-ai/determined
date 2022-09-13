import { useStore } from 'contexts/Store';
import {
  DetailedUser,
  ExperimentPermissionsArgs,
  ModelItem,
  ModelVersion,
  Permission,
  Project,
  ProjectExperiment,
  UserAssignment,
  UserRole,
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

interface PermissionWorkspace {
  id: number;
  userId?: number;
}

interface WorkspacePermissionsArgs {
  workspace?: PermissionWorkspace;
}

interface PermissionsHook {
  canDeleteExperiment: (arg0: ExperimentPermissionsArgs) => boolean;
  canDeleteModel: (arg0: ModelPermissionsArgs) => boolean;
  canDeleteModelVersion: (arg0: ModelVersionPermissionsArgs) => boolean;
  canDeleteProjects: (arg0: ProjectPermissionsArgs) => boolean;
  canDeleteWorkspace: (arg0: WorkspacePermissionsArgs) => boolean;
  canGetPermissions: () => boolean;
  canModifyProjects: (arg0: ProjectPermissionsArgs) => boolean;
  canModifyWorkspace: (arg0: WorkspacePermissionsArgs) => boolean;
  canMoveExperiment: (arg0: ExperimentPermissionsArgs) => boolean;
  canMoveProjects: (arg0: ProjectPermissionsArgs) => boolean;
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
  const canViewWorkspaces = relevantPermissions(userAssignments, userRoles).has('oss_user');

  return {
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
    canGetPermissions: () => canGetPermissions(user, userAssignments, userRoles),
    canModifyProjects: (args: ProjectPermissionsArgs) =>
      canModifyWorkspaceProjects(args.workspace, args.project, user, userAssignments, userRoles),
    canModifyWorkspace: (args: WorkspacePermissionsArgs) =>
      canModifyWorkspace(args.workspace, user, userAssignments, userRoles),
    canMoveExperiment: (args: ExperimentPermissionsArgs) =>
      canMoveExperiment(args.experiment, user, userAssignments, userRoles),
    canMoveProjects: (args: ProjectPermissionsArgs) =>
      canMoveWorkspaceProjects(args.workspace, args.project, user, userAssignments, userRoles),
    canViewWorkspaces,
  };
};

// Permissions inside this workspace scope (no workspace = cluster-wide scope)
const relevantPermissions = (
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[],
  workspaceId?: number
): Set<string> => {
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
      permissions = permissions.concat(r.permissions.filter((p) => p.globalOnly || workspaceId));
    });
  return new Set<string>(permissions.map((p) => p.name));
};

// Experiment actions
const canDeleteExperiment = (
  experiment: ProjectExperiment,
  user?: DetailedUser,
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[]
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
  userRoles?: UserRole[]
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

// User actions
const canGetPermissions = (
  user?: DetailedUser,
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[]
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles);
  return !!user && (permitted.has('oss_user') ? user.isAdmin : permitted.has('view_permissions'));
};

// Model and ModelVersion actions
const canDeleteModel = (
  model: ModelItem,
  user?: DetailedUser,
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[]
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
  userRoles?: UserRole[]
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
  userRoles?: UserRole[]
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
  userRoles?: UserRole[]
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
  userRoles?: UserRole[]
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
const canDeleteWorkspace = (
  workspace?: PermissionWorkspace,
  user?: DetailedUser,
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[]
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
  userRoles?: UserRole[]
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

export default usePermissions;
