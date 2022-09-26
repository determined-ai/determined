import { useStore } from 'contexts/Store';
import useFeature from 'hooks/useFeature';
import { V1PermissionType } from 'services/api-ts-sdk/api';
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
  rbacAllPermission: boolean;
  rbacEnabled: boolean;
  rbacReadPermission: boolean;
  user?: DetailedUser;
  userAssignments?: UserAssignment[];
  userRoles?: UserRole[];
}

interface PermissionWorkspace {
  id: number;
  userId?: number;
}

interface WorkspacePermissionsArgs {
  workspace?: PermissionWorkspace;
}

interface MovePermissionsArgs {
  destination?: PermissionWorkspace;
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
  canMoveExperimentsTo: (arg0: MovePermissionsArgs) => boolean;
  canMoveProjects: (arg0: ProjectPermissionsArgs) => boolean;
  canMoveProjectsTo: (arg0: MovePermissionsArgs) => boolean;
  canUpdateRoles: (arg0: ProjectPermissionsArgs) => boolean;
  canViewExperimentArtifacts: (arg0: WorkspacePermissionsArgs) => boolean;
  canViewGroups: boolean;
  canViewUsers: boolean;
  canViewWorkspace: (arg0: WorkspacePermissionsArgs) => boolean;
}

const usePermissions = (): PermissionsHook => {
  const {
    auth: { user },
    userAssignments,
    userRoles,
  } = useStore();
  const rbacEnabled = useFeature().isOn('rbac');
  const rbacAllPermission = useFeature().isOn('mock_permissions_all');
  const rbacReadPermission = useFeature().isOn('mock_permissions_read') || rbacAllPermission;
  const rbacOpts = {
    rbacAllPermission,
    rbacEnabled,
    rbacReadPermission,
    user,
    userAssignments,
    userRoles,
  };

<<<<<<< HEAD
  // Determine if the user has access to any workspaces
  // Should be updated to check user assignments and roles once available
  const canViewWorkspaces =
    !rbacEnabled ||
    rbacReadPermission ||
    relevantPermissions(userAssignments, userRoles).has('view_workspaces');

=======
>>>>>>> f492f26de (chore: convert permission.name string to permission.id enum)
  return {
    canAssignRoles: (args: WorkspacePermissionsArgs) => canAssignRoles(rbacOpts, args.workspace),
    canCreateExperiment: (args: WorkspacePermissionsArgs) =>
      canCreateExperiment(rbacOpts, args.workspace),
    canCreateProject: (args: WorkspacePermissionsArgs) =>
      canCreateProject(rbacOpts, args.workspace),
    canCreateWorkspace: canCreateWorkspace(rbacOpts),
    canDeleteExperiment: (args: ExperimentPermissionsArgs) =>
<<<<<<< HEAD
      canDeleteExperiment(rbacOpts, args.experiment),
    canDeleteModel: (args: ModelPermissionsArgs) => canDeleteModel(rbacOpts, args.model),
    canDeleteModelVersion: (args: ModelVersionPermissionsArgs) =>
      canDeleteModelVersion(rbacOpts, args.modelVersion),
=======
      canDeleteExperiment(args.experiment, user, userAssignments, userRoles),
    canDeleteModel: (args: ModelPermissionsArgs) => canDeleteModel(args.model, user),
    canDeleteModelVersion: (args: ModelVersionPermissionsArgs) =>
      canDeleteModelVersion(args.modelVersion, user),
>>>>>>> f492f26de (chore: convert permission.name string to permission.id enum)
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
<<<<<<< HEAD
      canModifyWorkspace(rbacOpts, args.workspace),
    canMoveExperiment: (args: ExperimentPermissionsArgs) =>
      canMoveExperiment(rbacOpts, args.experiment),
    canMoveProjects: (args: ProjectPermissionsArgs) =>
      canMoveWorkspaceProjects(rbacOpts, args.workspace, args.project),
    canUpdateRoles: (args: WorkspacePermissionsArgs) => canUpdateRoles(rbacOpts, args.workspace),
=======
      canModifyWorkspace(args.workspace, user, userAssignments, userRoles),
    canMoveExperiment: (args: ExperimentPermissionsArgs) =>
      canMoveExperiment(args.experiment, user, userAssignments, userRoles),
    canMoveExperimentsTo: (args: MovePermissionsArgs) =>
      canMoveExperimentsTo(args.destination, user, userAssignments, userRoles),
    canMoveProjects: (args: ProjectPermissionsArgs) =>
      canMoveWorkspaceProjects(args.project, user, userAssignments, userRoles),
    canMoveProjectsTo: (args: MovePermissionsArgs) =>
      canMoveProjectsTo(args.destination, user, userAssignments, userRoles),
    canUpdateRoles: (args: WorkspacePermissionsArgs) =>
      canUpdateRoles(args.workspace, user, userAssignments, userRoles),
>>>>>>> f492f26de (chore: convert permission.name string to permission.id enum)
    canViewExperimentArtifacts: (args: WorkspacePermissionsArgs) =>
      canViewExperimentArtifacts(rbacOpts, args.workspace),
    canViewGroups: canViewGroups(rbacOpts),
    canViewUsers: canAdministrateUsers(rbacOpts),
    canViewWorkspace: (args: WorkspacePermissionsArgs) =>
<<<<<<< HEAD
      canViewWorkspace(rbacOpts, args.workspace),
    canViewWorkspaces,
=======
      canViewWorkspace(args.workspace, userAssignments, userRoles),
>>>>>>> f492f26de (chore: convert permission.name string to permission.id enum)
  };
};

// Permissions inside this workspace scope (no workspace = cluster-wide scope)
// Typically returns a Set<string> of permissions.
const relevantPermissions = (
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[],
  workspaceId?: number,
): Set<V1PermissionType> => {
  if (!userAssignments || !userRoles) {
    // console.error('missing UserAssignment or UserRole');
    return new Set<V1PermissionType>();
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
  return new Set<V1PermissionType>(permissions.map((p) => p.id));
};

// User actions
const canAdministrateUsers = ({
  rbacAllPermission,
  rbacEnabled,
  user,
  userAssignments,
  userRoles,
}: RbacOptsProps): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles);
  return (
<<<<<<< HEAD
    rbacAllPermission ||
    (!!user && (rbacEnabled ? permitted.has('PERMISSION_CAN_ADMINISTRATE_USERS') : user.isAdmin))
  );
};

const canViewGroups = ({ rbacReadPermission, rbacEnabled, user }: RbacOptsProps): boolean => {
  return rbacReadPermission || (!!user && (rbacEnabled || user.isAdmin));
=======
    !!user &&
    (permitted.has(V1PermissionType.OSSUSER)
      ? user.isAdmin
      : permitted.has(V1PermissionType.ADMINISTRATEUSER))
  );
};

const canViewGroups = (
  user?: DetailedUser,
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[],
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles);
  return !!user && (permitted.has(V1PermissionType.OSSUSER) ? user.isAdmin : true);
>>>>>>> f492f26de (chore: convert permission.name string to permission.id enum)
};

const canModifyGroups = ({
  rbacAllPermission,
  rbacEnabled,
  user,
  userAssignments,
  userRoles,
}: RbacOptsProps): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles);
  return (
<<<<<<< HEAD
    rbacAllPermission ||
    (!!user && (rbacEnabled ? permitted.has('PERMISSION_CAN_UPDATE_GROUP') : user.isAdmin))
=======
    !!user &&
    (permitted.has(V1PermissionType.OSSUSER)
      ? user.isAdmin
      : permitted.has(V1PermissionType.UPDATEGROUP))
>>>>>>> f492f26de (chore: convert permission.name string to permission.id enum)
  );
};

// Experiment actions
const canCreateExperiment = (
  { rbacAllPermission, rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return !!workspace && (!rbacEnabled || rbacAllPermission || permitted.has('create_experiment'));
};

const canDeleteExperiment = (
  { rbacAllPermission, rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  experiment: ProjectExperiment,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, experiment.workspaceId);
  return (
<<<<<<< HEAD
    rbacAllPermission ||
    (!!experiment &&
      !!user &&
      (rbacEnabled
        ? permitted.has('delete_experiment')
        : user.isAdmin || user.id === experiment.userId))
=======
    !!experiment &&
    !!user &&
    (permitted.has(V1PermissionType.OSSUSER)
      ? user.isAdmin || user.id === experiment.userId
      : permitted.has(V1PermissionType.DELETEEXPERIMENT))
>>>>>>> f492f26de (chore: convert permission.name string to permission.id enum)
  );
};

const canModifyExperiment = (
  { rbacAllPermission, rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    rbacAllPermission || (!!workspace && (!rbacEnabled || permitted.has('update_experiments')))
  );
};

const canModifyExperimentMetadata = (
  { rbacAllPermission, rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    !!workspace &&
    (!rbacEnabled || rbacAllPermission || permitted.has('update_experiment_metadata'))
  );
};

const canMoveExperiment = (
  { rbacAllPermission, rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  experiment: ProjectExperiment,
<<<<<<< HEAD
<<<<<<< HEAD
=======
=======
  user?: DetailedUser,
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[],
): boolean => {
  // Destination is not needed when querying if an experiment can be moved from its source.
  const srcPermit = relevantPermissions(userAssignments, userRoles, experiment.workspaceId);
  return (
    !!user &&
    (srcPermit.has(V1PermissionType.OSSUSER)
      ? user.isAdmin || user.id === experiment.userId
      : srcPermit.has(V1PermissionType.DELETEEXPERIMENT))
  );
};

const canMoveExperimentsTo = (
>>>>>>> 37f4f48f2 (change rules for moving projects/experiments)
  destination?: PermissionWorkspace,
  user?: DetailedUser,
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[],
>>>>>>> f492f26de (chore: convert permission.name string to permission.id enum)
): boolean => {
  const destPermit = relevantPermissions(userAssignments, userRoles, destination?.id);
  return (
<<<<<<< HEAD
<<<<<<< HEAD
    rbacAllPermission ||
    (!!experiment &&
      !!user &&
      (rbacEnabled
        ? permitted.has('move_experiment')
        : user.isAdmin || user.id === experiment.userId))
=======
    !!experiment &&
    !!user &&
    (srcPermit.has(V1PermissionType.OSSUSER)
      ? user.isAdmin || user.id === experiment.userId
      : srcPermit.has(V1PermissionType.DELETEEXPERIMENT) &&
        (!destination || destPermit.has(V1PermissionType.CREATEEXPERIMENT)))
>>>>>>> f492f26de (chore: convert permission.name string to permission.id enum)
=======
    !!user &&
    (destPermit.has(V1PermissionType.OSSUSER) || destPermit.has(V1PermissionType.CREATEEXPERIMENT))
>>>>>>> 37f4f48f2 (change rules for moving projects/experiments)
  );
};

// experiment artifacts (checkpoints, metrics, etc.)
const canViewExperimentArtifacts = (
  { rbacEnabled, rbacReadPermission, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    !!workspace &&
    (!rbacEnabled || rbacReadPermission || permitted.has('view_experiment_artifacts'))
  );
};

// User actions
const canGetPermissions = ({
  rbacAllPermission,
  rbacEnabled,
  user,
  userAssignments,
  userRoles,
}: RbacOptsProps): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles);
  return (
    rbacAllPermission ||
    (!!user && (rbacEnabled ? permitted.has('view_permissions') : user.isAdmin))
  );
};

// Model and ModelVersion actions
const canDeleteModel = (
  { rbacAllPermission, rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  model: ModelItem,
<<<<<<< HEAD
=======
  user?: DetailedUser,
  // userAssignments?: UserAssignment[],
  // userRoles?: UserRole[],
>>>>>>> f492f26de (chore: convert permission.name string to permission.id enum)
): boolean => {
  // const permitted = relevantPermissions(userAssignments, userRoles);
  return (
<<<<<<< HEAD
    rbacAllPermission ||
    (!!model &&
      !!user &&
      (rbacEnabled ? permitted.has('delete_model') : user.isAdmin || user.id === model.userId))
=======
    !!model &&
    !!user &&
    // (permitted.has(V1PermissionType.OSSUSER) ?
    (user.isAdmin || user.id === model.userId)
    // : permitted.has('delete_model'))
>>>>>>> f492f26de (chore: convert permission.name string to permission.id enum)
  );
};

const canDeleteModelVersion = (
  { rbacAllPermission, rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  modelVersion?: ModelVersion,
<<<<<<< HEAD
=======
  user?: DetailedUser,
  // userAssignments?: UserAssignment[],
  // userRoles?: UserRole[],
>>>>>>> f492f26de (chore: convert permission.name string to permission.id enum)
): boolean => {
  // const permitted = relevantPermissions(userAssignments, userRoles);
  return (
<<<<<<< HEAD
    rbacAllPermission ||
    (!!modelVersion &&
      !!user &&
      (rbacEnabled
        ? permitted.has('delete_model_version')
        : user.isAdmin || user.id === modelVersion.userId))
=======
    !!modelVersion &&
    !!user &&
    // (permitted.has(V1PermissionType.OSSUSER) ?
    (user.isAdmin || user.id === modelVersion.userId)
    // : permitted.has('delete_model_version'))
>>>>>>> f492f26de (chore: convert permission.name string to permission.id enum)
  );
};

// Project actions
// Currently the smallest scope is workspace
const canCreateProject = (
  { rbacAllPermission, rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return !rbacEnabled || rbacAllPermission || permitted.has('create_project');
};

const canDeleteWorkspaceProjects = (
  { rbacAllPermission, rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
  project?: Project,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
<<<<<<< HEAD
    rbacAllPermission ||
    (!!workspace &&
      !!user &&
      !!project &&
      (rbacEnabled ? permitted.has('delete_projects') : user.isAdmin || user.id === project.userId))
=======
    !!workspace &&
    !!user &&
    !!project &&
    (permitted.has(V1PermissionType.OSSUSER)
      ? user.isAdmin || user.id === project.userId
      : permitted.has(V1PermissionType.DELETEPROJECT))
>>>>>>> f492f26de (chore: convert permission.name string to permission.id enum)
  );
};

const canModifyWorkspaceProjects = (
  { rbacAllPermission, rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
  project?: Project,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
<<<<<<< HEAD
    rbacAllPermission ||
    (!!workspace &&
      !!user &&
      !!project &&
      (rbacEnabled ? permitted.has('modify_projects') : user.isAdmin || user.id === project.userId))
=======
    !!workspace &&
    !!user &&
    !!project &&
    (permitted.has(V1PermissionType.OSSUSER)
      ? user.isAdmin || user.id === project.userId
      : permitted.has(V1PermissionType.UPDATEPROJECT))
>>>>>>> f492f26de (chore: convert permission.name string to permission.id enum)
  );
};

const canMoveWorkspaceProjects = (
<<<<<<< HEAD
  { rbacAllPermission, rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
  project?: Project,
=======
  project?: Project,
  user?: DetailedUser,
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[],
): boolean => {
  // can leave out project when we check for valid destinations
  const srcPermit = relevantPermissions(userAssignments, userRoles, project?.workspaceId);
  return (
    !!user &&
    (srcPermit.has(V1PermissionType.OSSUSER)
      ? !project || user.isAdmin || user.id === project.userId
      : srcPermit.has(V1PermissionType.DELETEPROJECT))
  );
};

const canMoveProjectsTo = (
  destination?: PermissionWorkspace,
  user?: DetailedUser,
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[],
>>>>>>> f492f26de (chore: convert permission.name string to permission.id enum)
): boolean => {
  const destPermit = relevantPermissions(userAssignments, userRoles, destination?.id);
  return (
<<<<<<< HEAD
    rbacAllPermission ||
    (!!workspace &&
      !!user &&
      !!project &&
      (rbacEnabled ? permitted.has('move_projects') : user.isAdmin || user.id === project.userId))
=======
    !!user &&
<<<<<<< HEAD
    !!project &&
    (srcPermit.has(V1PermissionType.OSSUSER)
      ? user.isAdmin || user.id === project.userId
      : srcPermit.has(V1PermissionType.DELETEPROJECT) &&
        (!destination || destPermit.has(V1PermissionType.CREATEPROJECT)))
>>>>>>> f492f26de (chore: convert permission.name string to permission.id enum)
=======
    (destPermit.has(V1PermissionType.OSSUSER) || destPermit.has(V1PermissionType.CREATEPROJECT))
>>>>>>> 37f4f48f2 (change rules for moving projects/experiments)
  );
};

// Workspace actions
const canCreateWorkspace = ({
  rbacAllPermission,
  rbacEnabled,
  userAssignments,
  userRoles,
}: RbacOptsProps): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles);
<<<<<<< HEAD
<<<<<<< HEAD
  return !rbacEnabled || rbacAllPermission || permitted.has('create_workspace');
=======
  return (
    permitted.has(V1PermissionType.OSSUSER) || permitted.has(V1PermissionType.CREATEWORKSPACE)
  );
>>>>>>> f492f26de (chore: convert permission.name string to permission.id enum)
=======
  return permitted.has(V1PermissionType.OSSUSER) || permitted.has(V1PermissionType.CREATEWORKSPACE);
>>>>>>> 37f4f48f2 (change rules for moving projects/experiments)
};

const canDeleteWorkspace = (
  { rbacAllPermission, rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
<<<<<<< HEAD
    rbacAllPermission ||
    (!!workspace &&
      !!user &&
      (rbacEnabled
        ? permitted.has('delete_workspace')
        : user.isAdmin || user.id === workspace.userId))
=======
    !!workspace &&
    !!user &&
    (permitted.has(V1PermissionType.OSSUSER)
      ? user.isAdmin || user.id === workspace.userId
      : permitted.has(V1PermissionType.DELETEWORKSPACE))
>>>>>>> f492f26de (chore: convert permission.name string to permission.id enum)
  );
};

const canModifyWorkspace = (
  { rbacAllPermission, rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
<<<<<<< HEAD
    rbacAllPermission ||
    (!!workspace &&
      !!user &&
      (rbacEnabled
        ? permitted.has('modify_workspace')
        : user.isAdmin || user.id === workspace.userId))
=======
    !!workspace &&
    !!user &&
    (permitted.has(V1PermissionType.OSSUSER)
      ? user.isAdmin || user.id === workspace.userId
      : permitted.has(V1PermissionType.UPDATEWORKSPACE))
>>>>>>> f492f26de (chore: convert permission.name string to permission.id enum)
  );
};

const canViewWorkspace = (
  { rbacEnabled, rbacReadPermission, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
<<<<<<< HEAD
  return !!workspace && (!rbacEnabled || rbacReadPermission || permitted.has('view_workspace'));
=======
  return (
    !!workspace &&
    (permitted.has(V1PermissionType.OSSUSER) || permitted.has(V1PermissionType.VIEWWORKSPACE))
  );
>>>>>>> f492f26de (chore: convert permission.name string to permission.id enum)
};

const canUpdateRoles = (
  { rbacAllPermission, rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    rbacAllPermission ||
    (!!workspace &&
      !!user &&
      (rbacEnabled ? permitted.has('update_roles') : user.isAdmin || user.id === workspace.userId))
  );
};

const canAssignRoles = (
  { rbacAllPermission, rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    rbacAllPermission ||
    (!!workspace &&
      !!user &&
      (rbacEnabled ? permitted.has('assign_roles') : user.isAdmin || user.id === workspace.userId))
  );
};

export default usePermissions;
