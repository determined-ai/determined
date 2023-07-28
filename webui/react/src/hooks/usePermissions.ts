import { useObservable } from 'micro-observables';
import { useMemo } from 'react';

import { V1PermissionType } from 'services/api-ts-sdk/api';
import determinedStore from 'stores/determinedInfo';
import permissionStore from 'stores/permissions';
import userStore from 'stores/users';
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
import { Loadable } from 'utils/loadable';

interface ModelPermissionsArgs {
  model: ModelItem;
}

interface ModelWorkspacePermissionsArgs {
  workspaceId: number;
}

interface ModelVersionPermissionsArgs {
  modelVersion: ModelVersion;
}

interface ProjectPermissionsArgs {
  project?: Project;
  workspace?: PermissionWorkspace;
}

interface RbacOptsProps {
  currentUser?: DetailedUser;
  rbacEnabled: boolean;
  userAssignments?: UserAssignment[];
  userRoles?: UserRole[];
}

interface MovePermissionsArgs {
  destination?: PermissionWorkspace;
}

interface PermissionsHook {
  canAdministrateUsers: boolean;
  canAssignRoles: (arg0: WorkspacePermissionsArgs) => boolean;
  canCreateExperiment: (arg0: WorkspacePermissionsArgs) => boolean;
  canCreateModelVersion: (arg0: ModelPermissionsArgs) => boolean;
  canCreateModelWorkspace: (arg0: ModelWorkspacePermissionsArgs) => boolean;
  canCreateModels: boolean;
  canCreateNSC: boolean;
  canCreateProject: (arg0: WorkspacePermissionsArgs) => boolean;
  canCreateWorkspace: boolean;
  canCreateWorkspaceNSC(arg0: WorkspacePermissionsArgs): boolean;
  canDeleteExperiment: (arg0: ExperimentPermissionsArgs) => boolean;
  canDeleteModel: (arg0: ModelPermissionsArgs) => boolean;
  canDeleteModelVersion: (arg0: ModelVersionPermissionsArgs) => boolean;
  canDeleteProjects: (arg0: ProjectPermissionsArgs) => boolean;
  canDeleteWorkspace: (arg0: WorkspacePermissionsArgs) => boolean;
  canEditWebhooks: boolean;
  canManageResourcePoolBindings: boolean;
  canModifyExperiment: (arg0: WorkspacePermissionsArgs) => boolean;
  canModifyExperimentMetadata: (arg0: WorkspacePermissionsArgs) => boolean;
  canModifyGroups: boolean;
  canModifyModel: (arg0: ModelPermissionsArgs) => boolean;
  canModifyModelVersion: (arg0: ModelVersionPermissionsArgs) => boolean;
  canModifyPermissions: boolean;
  canModifyProjects: (arg0: ProjectPermissionsArgs) => boolean;
  canModifyUsers: boolean;
  canModifyWorkspace: (arg0: WorkspacePermissionsArgs) => boolean;
  canModifyWorkspaceAgentUserGroup: (arg0: WorkspacePermissionsArgs) => boolean;
  canModifyWorkspaceCheckpointStorage: (arg0: WorkspacePermissionsArgs) => boolean;
  canModifyWorkspaceNSC(arg0: WorkspacePermissionsArgs): boolean;
  canMoveExperiment: (arg0: ExperimentPermissionsArgs) => boolean;
  canMoveExperimentsTo: (arg0: MovePermissionsArgs) => boolean;
  canMoveModel: (arg0: MovePermissionsArgs) => boolean;
  canMoveProjects: (arg0: ProjectPermissionsArgs) => boolean;
  canMoveProjectsTo: (arg0: MovePermissionsArgs) => boolean;
  canUpdateRoles: (arg0: ProjectPermissionsArgs) => boolean;
  canViewExperimentArtifacts: (arg0: WorkspacePermissionsArgs) => boolean;
  canViewGroups: boolean;
  canViewModelRegistry: (arg0: WorkspacePermissionsArgs) => boolean;
  canViewWorkspace: (arg0: WorkspacePermissionsArgs) => boolean;
  canViewWorkspaces: boolean;
  loading: boolean;
}

const usePermissions = (): PermissionsHook => {
  const { rbacEnabled } = useObservable(determinedStore.info);
  const loadableCurrentUser = useObservable(userStore.currentUser);
  const currentUser = Loadable.getOrElse(undefined, loadableCurrentUser);

  // Loadables keep track of loading status
  // userAssignments and userRoles should always be an array -- empty arrays until loading is complete.
  const loadablePermissions = useObservable(permissionStore.permissions);
  const myAssignments = Loadable.getOrElse([], useObservable(permissionStore.myAssignments));
  const myRoles = Loadable.getOrElse([], useObservable(permissionStore.myRoles));

  const rbacOpts = useMemo(
    () => ({
      currentUser,
      rbacEnabled,
      userAssignments: myAssignments,
      userRoles: myRoles,
    }),
    [currentUser, myAssignments, myRoles, rbacEnabled],
  );

  const permissions = useMemo(
    () => ({
      canAdministrateUsers: canAdministrateUsers(rbacOpts),
      canAssignRoles: (args: WorkspacePermissionsArgs) => canAssignRoles(rbacOpts, args.workspace),
      canCreateExperiment: (args: WorkspacePermissionsArgs) =>
        canCreateExperiment(rbacOpts, args.workspace),
      canCreateModels: canCreateModels(rbacOpts),
      canCreateModelVersion: (args: ModelPermissionsArgs) =>
        canCreateModelVersion(rbacOpts, args.model),
      canCreateModelWorkspace: (args: ModelWorkspacePermissionsArgs) =>
        canCreateModelWorkspace(rbacOpts, args.workspaceId),
      canCreateNSC: canCreateNSC(rbacOpts),
      canCreateProject: (args: WorkspacePermissionsArgs) =>
        canCreateProject(rbacOpts, args.workspace),
      canCreateWorkspace: canCreateWorkspace(rbacOpts),
      canCreateWorkspaceNSC: (args: WorkspacePermissionsArgs) =>
        canCreateWorkspaceNSC(rbacOpts, args.workspace),
      canDeleteExperiment: (args: ExperimentPermissionsArgs) =>
        canDeleteExperiment(rbacOpts, args.experiment),
      canDeleteModel: (args: ModelPermissionsArgs) => canDeleteModel(rbacOpts, args.model),
      canDeleteModelVersion: (args: ModelVersionPermissionsArgs) =>
        canDeleteModelVersion(rbacOpts, args.modelVersion),
      canDeleteProjects: (args: ProjectPermissionsArgs) =>
        canDeleteWorkspaceProjects(rbacOpts, args.workspace, args.project),
      canDeleteWorkspace: (args: WorkspacePermissionsArgs) =>
        canDeleteWorkspace(rbacOpts, args.workspace),
      canEditWebhooks: canEditWebhooks(rbacOpts),
      canManageResourcePoolBindings: canManageResourcePoolBindings(rbacOpts),
      canModifyExperiment: (args: WorkspacePermissionsArgs) =>
        canModifyExperiment(rbacOpts, args.workspace),
      canModifyExperimentMetadata: (args: WorkspacePermissionsArgs) =>
        canModifyExperimentMetadata(rbacOpts, args.workspace),
      canModifyGroups: canModifyGroups(rbacOpts),
      canModifyModel: (args: ModelPermissionsArgs) => canModifyModel(rbacOpts, args.model),
      canModifyModelVersion: (args: ModelVersionPermissionsArgs) =>
        canModifyModelVersion(rbacOpts, args.modelVersion),
      canModifyPermissions: canModifyPermissions(rbacOpts),
      canModifyProjects: (args: ProjectPermissionsArgs) =>
        canModifyWorkspaceProjects(rbacOpts, args.workspace, args.project),
      canModifyUsers: canAdministrateUsers(rbacOpts),
      canModifyWorkspace: (args: WorkspacePermissionsArgs) =>
        canModifyWorkspace(rbacOpts, args.workspace),
      canModifyWorkspaceAgentUserGroup: (args: WorkspacePermissionsArgs) =>
        canModifyWorkspaceAgentUserGroup(rbacOpts, args.workspace),
      canModifyWorkspaceCheckpointStorage: (args: WorkspacePermissionsArgs) =>
        canModifyWorkspaceCheckpointStorage(rbacOpts, args.workspace),
      canModifyWorkspaceNSC: (args: WorkspacePermissionsArgs) =>
        canModifyWorkspaceNSC(rbacOpts, args.workspace),
      canMoveExperiment: (args: ExperimentPermissionsArgs) =>
        canMoveExperiment(rbacOpts, args.experiment),
      canMoveExperimentsTo: (args: MovePermissionsArgs) =>
        canMoveExperimentsTo(rbacOpts, args.destination),
      canMoveModel: (args: MovePermissionsArgs) => canMoveModel(rbacOpts, args.destination),
      canMoveProjects: (args: ProjectPermissionsArgs) =>
        canMoveWorkspaceProjects(rbacOpts, args.project),
      canMoveProjectsTo: (args: MovePermissionsArgs) =>
        canMoveProjectsTo(rbacOpts, args.destination),
      canUpdateRoles: (args: WorkspacePermissionsArgs) => canUpdateRoles(rbacOpts, args.workspace),
      canViewExperimentArtifacts: (args: WorkspacePermissionsArgs) =>
        canViewExperimentArtifacts(rbacOpts, args.workspace),
      canViewGroups: canViewGroups(rbacOpts),
      canViewModelRegistry: (args: WorkspacePermissionsArgs) =>
        canViewModelRegistry(rbacOpts, args.workspace),
      canViewWorkspace: (args: WorkspacePermissionsArgs) =>
        canViewWorkspace(rbacOpts, args.workspace),
      canViewWorkspaces: canViewWorkspaces(rbacOpts),
      loading:
        rbacOpts.rbacEnabled &&
        Loadable.isLoading(Loadable.all([loadableCurrentUser, loadablePermissions])),
    }),
    [rbacOpts, loadableCurrentUser, loadablePermissions],
  );

  return permissions;
};

// Permissions inside this workspace scope (no workspace = cluster-wide scope)
// Typically returns a Set<string> of permissions.
const relevantPermissions = (
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[],
  workspaceId?: number,
): Set<V1PermissionType> => {
  if (!userAssignments || !userRoles) {
    return new Set<V1PermissionType>();
  }
  const relevantAssigned = userAssignments
    .filter(
      (a) =>
        a.scopeCluster ||
        (workspaceId &&
          a.workspaces &&
          Array.from(Object.values(a.workspaces)).includes(workspaceId)),
    )
    .map((a) => a.roleId);
  let permissions = Array<Permission>();
  userRoles
    .filter((r) => relevantAssigned.includes(r.id))
    .forEach((r) => {
      permissions = permissions.concat(r.permissions);
    });
  return new Set<V1PermissionType>(permissions.map((p) => p.id));
};

// User actions
const canAdministrateUsers = ({
  currentUser,
  rbacEnabled,
  userAssignments,
  userRoles,
}: RbacOptsProps): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles);
  return (
    !!currentUser &&
    (rbacEnabled ? permitted.has(V1PermissionType.ADMINISTRATEUSER) : currentUser.isAdmin)
  );
};

const canViewGroups = ({ currentUser, rbacEnabled }: RbacOptsProps): boolean => {
  return !!currentUser && (rbacEnabled || currentUser.isAdmin);
};

const canViewModelRegistry = (
  { rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  // For OSS, everyone can view model registry
  // For RBAC, users with VIEWMODELREGISTRY permission can view model registry
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return !rbacEnabled || permitted.has(V1PermissionType.VIEWMODELREGISTRY);
};

const canCreateModelWorkspace = (
  { rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspaceId: number,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspaceId);
  return !rbacEnabled || permitted.has(V1PermissionType.CREATEMODELREGISTRY);
};

const canCreateModels = ({ rbacEnabled, userRoles }: RbacOptsProps): boolean => {
  return (
    !rbacEnabled ||
    (!!userRoles &&
      !!userRoles.find(
        (r) => !!r.permissions.find((p) => p.id === V1PermissionType.CREATEMODELREGISTRY),
      ))
  );
};

const canModifyGroups = ({
  currentUser,
  rbacEnabled,
  userAssignments,
  userRoles,
}: RbacOptsProps): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles);
  return (
    !!currentUser &&
    (rbacEnabled ? permitted.has(V1PermissionType.UPDATEGROUP) : currentUser.isAdmin)
  );
};

const canModifyPermissions = ({
  rbacEnabled,
  userAssignments,
  userRoles,
}: RbacOptsProps): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles);
  return rbacEnabled && permitted.has(V1PermissionType.ADMINISTRATEUSER);
};

// Experiment actions
const canCreateExperiment = (
  { rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return !!workspace && (!rbacEnabled || permitted.has(V1PermissionType.CREATEEXPERIMENT));
};

const canDeleteExperiment = (
  { currentUser, rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  experiment: ProjectExperiment,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, experiment.workspaceId);
  return (
    !!experiment &&
    !!currentUser &&
    (rbacEnabled
      ? permitted.has(V1PermissionType.DELETEEXPERIMENT)
      : currentUser.isAdmin || currentUser.id === experiment.userId)
  );
};

const canModifyExperiment = (
  { rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return !!workspace && (!rbacEnabled || permitted.has(V1PermissionType.UPDATEEXPERIMENT));
};

const canModifyExperimentMetadata = (
  { rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return !!workspace && (!rbacEnabled || permitted.has(V1PermissionType.UPDATEEXPERIMENTMETADATA));
};

const canMoveExperiment = (
  { currentUser, rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  experiment: ProjectExperiment,
): boolean => {
  const srcPermit = relevantPermissions(userAssignments, userRoles, experiment.workspaceId);
  return (
    !!currentUser &&
    (rbacEnabled
      ? srcPermit.has(V1PermissionType.DELETEEXPERIMENT)
      : currentUser.isAdmin || currentUser.id === experiment.userId)
  );
};

const canMoveExperimentsTo = (
  { currentUser, rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  destination?: PermissionWorkspace,
): boolean => {
  const destPermit = relevantPermissions(userAssignments, userRoles, destination?.id);
  return !!currentUser && (!rbacEnabled || destPermit.has(V1PermissionType.CREATEEXPERIMENT));
};

// experiment artifacts (checkpoints, metrics, etc.)
const canViewExperimentArtifacts = (
  { rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return !!workspace && (!rbacEnabled || permitted.has(V1PermissionType.VIEWEXPERIMENTARTIFACTS));
};

// Model and ModelVersion actions
const canDeleteModel = (
  { currentUser, rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  model: ModelItem,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, model.workspaceId);
  return rbacEnabled
    ? permitted.has(V1PermissionType.DELETEMODELREGISTRY)
    : !!currentUser && (currentUser.isAdmin || currentUser.id === model?.userId);
};

const canModifyModel = (
  { rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  model: ModelItem,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, model.workspaceId);
  return !rbacEnabled || permitted.has(V1PermissionType.EDITMODELREGISTRY);
};

const canCreateModelVersion = (
  { rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  model: ModelItem,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, model.workspaceId);
  return !rbacEnabled || permitted.has(V1PermissionType.CREATEMODELREGISTRY);
};

const canDeleteModelVersion = (
  { currentUser, rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  modelVersion: ModelVersion,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, modelVersion.model.workspaceId);
  return rbacEnabled
    ? permitted.has(V1PermissionType.DELETEMODELREGISTRY)
    : !!currentUser && (currentUser.isAdmin || currentUser.id === modelVersion?.userId);
};

const canModifyModelVersion = (
  { rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  modelVersion: ModelVersion,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, modelVersion.model.workspaceId);
  return !rbacEnabled || permitted.has(V1PermissionType.EDITMODELREGISTRY);
};

// Project actions
// Currently the smallest scope is workspace
const canCreateProject = (
  { rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return !rbacEnabled || permitted.has(V1PermissionType.CREATEPROJECT);
};

const canDeleteWorkspaceProjects = (
  { currentUser, rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
  project?: Project,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    !!workspace &&
    !!currentUser &&
    !!project &&
    (rbacEnabled
      ? permitted.has(V1PermissionType.DELETEPROJECT)
      : currentUser.isAdmin || currentUser.id === project.userId)
  );
};

const canModifyWorkspaceProjects = (
  { currentUser, rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
  project?: Project,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    !!workspace &&
    !!currentUser &&
    !!project &&
    (rbacEnabled
      ? permitted.has(V1PermissionType.UPDATEPROJECT)
      : currentUser.isAdmin || currentUser.id === project.userId)
  );
};

const canMoveWorkspaceProjects = (
  { currentUser, rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  project?: Project,
): boolean => {
  const srcPermit = relevantPermissions(userAssignments, userRoles, project?.workspaceId);
  return (
    !!currentUser &&
    !!project &&
    (rbacEnabled
      ? srcPermit.has(V1PermissionType.DELETEPROJECT)
      : currentUser.isAdmin || currentUser.id === project.userId)
  );
};

const canMoveProjectsTo = (
  { currentUser, rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  destination?: PermissionWorkspace,
): boolean => {
  const destPermit = relevantPermissions(userAssignments, userRoles, destination?.id);
  return !!currentUser && (!rbacEnabled || destPermit.has(V1PermissionType.CREATEPROJECT));
};

const canMoveModel = (
  { rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  destination?: PermissionWorkspace,
): boolean => {
  const destPermit = relevantPermissions(userAssignments, userRoles, destination?.id);
  return !rbacEnabled || destPermit.has(V1PermissionType.CREATEMODELREGISTRY);
};

// Workspace actions
const canCreateWorkspace = ({
  rbacEnabled,
  userAssignments,
  userRoles,
}: RbacOptsProps): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles);
  return !rbacEnabled || permitted.has(V1PermissionType.CREATEWORKSPACE);
};

const canDeleteWorkspace = (
  { currentUser, rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    !!workspace &&
    !!currentUser &&
    (rbacEnabled
      ? permitted.has(V1PermissionType.DELETEWORKSPACE)
      : currentUser.isAdmin || currentUser.id === workspace.userId)
  );
};

const canModifyWorkspace = (
  { currentUser, rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    !!workspace &&
    !!currentUser &&
    (rbacEnabled
      ? permitted.has(V1PermissionType.UPDATEWORKSPACE)
      : currentUser.isAdmin || currentUser.id === workspace.userId)
  );
};

const canModifyWorkspaceAgentUserGroup = (
  { currentUser, rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    !!currentUser &&
    (rbacEnabled ? permitted.has(V1PermissionType.SETWORKSPACEAGENTUSERGROUP) : currentUser.isAdmin)
  );
};

const canModifyWorkspaceCheckpointStorage = (
  { currentUser, rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    !!currentUser &&
    (rbacEnabled
      ? permitted.has(V1PermissionType.SETWORKSPACECHECKPOINTSTORAGECONFIG)
      : currentUser.isAdmin)
  );
};

const canViewWorkspace = (
  { rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return !!workspace && (!rbacEnabled || permitted.has(V1PermissionType.VIEWWORKSPACE));
};

const canViewWorkspaces = ({ rbacEnabled, userRoles }: RbacOptsProps): boolean => {
  return (
    !rbacEnabled ||
    (!!userRoles &&
      !!userRoles.find((r) => !!r.permissions.find((p) => p.id === V1PermissionType.VIEWWORKSPACE)))
  );
};

const canUpdateRoles = (
  { currentUser, rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    !!workspace &&
    !!currentUser &&
    (rbacEnabled
      ? permitted.has(V1PermissionType.UPDATEROLES)
      : currentUser.isAdmin || currentUser.id === workspace.userId)
  );
};

const canAssignRoles = (
  { currentUser, rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    (!!currentUser && !!workspace && currentUser.id === workspace.userId) ||
    (!!currentUser &&
      (rbacEnabled ? permitted.has(V1PermissionType.ASSIGNROLES) : currentUser.isAdmin))
  );
};

const canCreateNSC = ({ rbacEnabled, userRoles }: RbacOptsProps): boolean => {
  return (
    !rbacEnabled ||
    (!!userRoles &&
      !!userRoles.find((r) => !!r.permissions.find((p) => p.id === V1PermissionType.CREATENSC)))
  );
};

const canCreateWorkspaceNSC = (
  { rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return !!workspace && (!rbacEnabled || permitted.has(V1PermissionType.CREATENSC));
};

const canModifyWorkspaceNSC = (
  { rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return !rbacEnabled || permitted.has(V1PermissionType.UPDATENSC);
};

/* Webhooks */

const canEditWebhooks = ({
  currentUser,
  rbacEnabled,
  userAssignments,
  userRoles,
}: RbacOptsProps): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles);
  return rbacEnabled
    ? permitted.has(V1PermissionType.EDITWEBHOOKS)
    : !!currentUser && currentUser.isAdmin;
};

/* Resource Pools */

const canManageResourcePoolBindings = ({
  currentUser,
  rbacEnabled,
  userAssignments,
  userRoles,
}: RbacOptsProps): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles);
  return rbacEnabled
    ? permitted.has(V1PermissionType.UPDATEMASTERCONFIG)
    : !!currentUser && currentUser.isAdmin;
};

export default usePermissions;
