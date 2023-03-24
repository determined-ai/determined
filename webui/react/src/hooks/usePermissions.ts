import { useObservable } from 'micro-observables';
import { useMemo } from 'react';

import useFeature from 'hooks/useFeature';
import { V1PermissionType } from 'services/api-ts-sdk/api';
import { PermissionsStore } from 'stores/permissions';
import usersStore from 'stores/users';
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
  rbacAllPermission: boolean;
  rbacEnabled: boolean;
  rbacReadPermission: boolean;
  user?: DetailedUser;
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
  const rbacEnabled = useFeature().isOn('rbac'),
    rbacAllPermission = useFeature().isOn('mock_permissions_all'),
    rbacReadPermission = useFeature().isOn('mock_permissions_read') || rbacAllPermission;

  const loadableCurrentUser = useObservable<Loadable<DetailedUser>>(usersStore.getCurrentUser());
  const user = Loadable.match(loadableCurrentUser, {
    Loaded: (cUser) => cUser,
    NotLoaded: () => undefined,
  });

  // Loadables keep track of loading status
  // userAssignments and userRoles should always be an array -- empty arrays until loading is complete.
  const loadableUserAssignments = useObservable<Loadable<UserAssignment[]>>(
    PermissionsStore.getMyAssignments(),
  );
  const userAssignments = Loadable.match(loadableUserAssignments, {
    Loaded: (uAssignments) => uAssignments,
    NotLoaded: () => [],
  });
  const loadableUserRoles = useObservable<Loadable<UserRole[]>>(PermissionsStore.getMyRoles());
  const userRoles = Loadable.match(loadableUserRoles, {
    Loaded: (uRoles) => uRoles,
    NotLoaded: () => [],
  });

  const rbacOpts = useMemo(
    () => ({
      rbacAllPermission,
      rbacEnabled,
      rbacReadPermission,
      user,
      userAssignments,
      userRoles,
    }),
    [rbacAllPermission, rbacEnabled, rbacReadPermission, user, userAssignments, userRoles],
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
        (Loadable.isLoading(loadableCurrentUser) ||
          Loadable.isLoading(loadableUserAssignments) ||
          Loadable.isLoading(loadableUserRoles)),
    }),
    [rbacOpts, loadableUserAssignments, loadableUserRoles, loadableCurrentUser],
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
  rbacAllPermission,
  rbacEnabled,
  user,
  userAssignments,
  userRoles,
}: RbacOptsProps): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles);
  return (
    rbacAllPermission ||
    (!!user && (rbacEnabled ? permitted.has(V1PermissionType.ADMINISTRATEUSER) : user.isAdmin))
  );
};

const canViewGroups = ({ rbacReadPermission, rbacEnabled, user }: RbacOptsProps): boolean => {
  return rbacReadPermission || (!!user && (rbacEnabled || user.isAdmin));
};

const canViewModelRegistry = (
  { rbacReadPermission, rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  // For OSS, everyone can view model registry
  // For RBAC, users with rbacReadPermission or VIEWMODELREGISTRY permission can view model resgistry
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return !rbacEnabled || rbacReadPermission || permitted.has(V1PermissionType.VIEWMODELREGISTRY);
};

const canCreateModelWorkspace = (
  { rbacReadPermission, rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspaceId: number,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspaceId);
  return !rbacEnabled || rbacReadPermission || permitted.has(V1PermissionType.CREATEMODELREGISTRY);
};

const canCreateModels = ({
  rbacReadPermission,
  rbacEnabled,
  userRoles,
}: RbacOptsProps): boolean => {
  return (
    !rbacEnabled ||
    rbacReadPermission ||
    (!!userRoles &&
      !!userRoles.find(
        (r) => !!r.permissions.find((p) => p.id === V1PermissionType.CREATEMODELREGISTRY),
      ))
  );
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
    rbacAllPermission ||
    (!!user && (rbacEnabled ? permitted.has(V1PermissionType.UPDATEGROUP) : user.isAdmin))
  );
};

const canModifyPermissions = ({
  rbacAllPermission,
  rbacEnabled,
  userAssignments,
  userRoles,
}: RbacOptsProps): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles);
  return rbacAllPermission || (rbacEnabled && permitted.has(V1PermissionType.ADMINISTRATEUSER));
};

// Experiment actions
const canCreateExperiment = (
  { rbacAllPermission, rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    !!workspace &&
    (!rbacEnabled || rbacAllPermission || permitted.has(V1PermissionType.CREATEEXPERIMENT))
  );
};

const canDeleteExperiment = (
  { rbacAllPermission, rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  experiment: ProjectExperiment,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, experiment.workspaceId);
  return (
    rbacAllPermission ||
    (!!experiment &&
      !!user &&
      (rbacEnabled
        ? permitted.has(V1PermissionType.DELETEEXPERIMENT)
        : user.isAdmin || user.id === experiment.userId))
  );
};

const canModifyExperiment = (
  { rbacAllPermission, rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    rbacAllPermission ||
    (!!workspace && (!rbacEnabled || permitted.has(V1PermissionType.UPDATEEXPERIMENT)))
  );
};

const canModifyExperimentMetadata = (
  { rbacAllPermission, rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    !!workspace &&
    (!rbacEnabled || rbacAllPermission || permitted.has(V1PermissionType.UPDATEEXPERIMENTMETADATA))
  );
};

const canMoveExperiment = (
  { rbacAllPermission, rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  experiment: ProjectExperiment,
): boolean => {
  const srcPermit = relevantPermissions(userAssignments, userRoles, experiment.workspaceId);
  return (
    rbacAllPermission ||
    (!!user &&
      (rbacEnabled
        ? srcPermit.has(V1PermissionType.DELETEEXPERIMENT)
        : user.isAdmin || user.id === experiment.userId))
  );
};

const canMoveExperimentsTo = (
  { rbacAllPermission, rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  destination?: PermissionWorkspace,
): boolean => {
  const destPermit = relevantPermissions(userAssignments, userRoles, destination?.id);
  return (
    rbacAllPermission ||
    (!!user && (!rbacEnabled || destPermit.has(V1PermissionType.CREATEEXPERIMENT)))
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
    (!rbacEnabled || rbacReadPermission || permitted.has(V1PermissionType.VIEWEXPERIMENTARTIFACTS))
  );
};

// Model and ModelVersion actions
const canDeleteModel = (
  { rbacAllPermission, rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  model: ModelItem,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, model.workspaceId);
  return (
    rbacAllPermission ||
    (rbacEnabled
      ? permitted.has(V1PermissionType.EDITMODELREGISTRY)
      : !!user && (user.isAdmin || user.id === model?.userId))
  );
};

const canModifyModel = (
  { rbacAllPermission, rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  model: ModelItem,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, model.workspaceId);
  return rbacAllPermission || !rbacEnabled || permitted.has(V1PermissionType.EDITMODELREGISTRY);
};

const canCreateModelVersion = (
  { rbacAllPermission, rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  model: ModelItem,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, model.workspaceId);
  return rbacAllPermission || !rbacEnabled || permitted.has(V1PermissionType.CREATEMODELREGISTRY);
};

const canDeleteModelVersion = (
  { rbacAllPermission, rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  modelVersion: ModelVersion,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, modelVersion.model.workspaceId);
  return (
    rbacAllPermission ||
    (rbacEnabled
      ? permitted.has(V1PermissionType.EDITMODELREGISTRY)
      : !!user && (user.isAdmin || user.id === modelVersion?.userId))
  );
};

const canModifyModelVersion = (
  { rbacAllPermission, rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  modelVersion: ModelVersion,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, modelVersion.model.workspaceId);
  return !rbacEnabled || rbacAllPermission || permitted.has(V1PermissionType.EDITMODELREGISTRY);
};

// Project actions
// Currently the smallest scope is workspace
const canCreateProject = (
  { rbacAllPermission, rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return !rbacEnabled || rbacAllPermission || permitted.has(V1PermissionType.CREATEPROJECT);
};

const canDeleteWorkspaceProjects = (
  { rbacAllPermission, rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
  project?: Project,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    rbacAllPermission ||
    (!!workspace &&
      !!user &&
      !!project &&
      (rbacEnabled
        ? permitted.has(V1PermissionType.DELETEPROJECT)
        : user.isAdmin || user.id === project.userId))
  );
};

const canModifyWorkspaceProjects = (
  { rbacAllPermission, rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
  project?: Project,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    rbacAllPermission ||
    (!!workspace &&
      !!user &&
      !!project &&
      (rbacEnabled
        ? permitted.has(V1PermissionType.UPDATEPROJECT)
        : user.isAdmin || user.id === project.userId))
  );
};

const canMoveWorkspaceProjects = (
  { rbacAllPermission, rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  project?: Project,
): boolean => {
  const srcPermit = relevantPermissions(userAssignments, userRoles, project?.workspaceId);
  return (
    rbacAllPermission ||
    (!!user &&
      !!project &&
      (rbacEnabled
        ? srcPermit.has(V1PermissionType.DELETEPROJECT)
        : user.isAdmin || user.id === project.userId))
  );
};

const canMoveProjectsTo = (
  { rbacAllPermission, rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  destination?: PermissionWorkspace,
): boolean => {
  const destPermit = relevantPermissions(userAssignments, userRoles, destination?.id);
  return (
    rbacAllPermission ||
    (!!user && (!rbacEnabled || destPermit.has(V1PermissionType.CREATEPROJECT)))
  );
};

const canMoveModel = (
  { rbacAllPermission, rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  destination?: PermissionWorkspace,
): boolean => {
  const destPermit = relevantPermissions(userAssignments, userRoles, destination?.id);
  return rbacAllPermission || !rbacEnabled || destPermit.has(V1PermissionType.CREATEMODELREGISTRY);
};

// Workspace actions
const canCreateWorkspace = ({
  rbacAllPermission,
  rbacEnabled,
  userAssignments,
  userRoles,
}: RbacOptsProps): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles);
  return !rbacEnabled || rbacAllPermission || permitted.has(V1PermissionType.CREATEWORKSPACE);
};

const canDeleteWorkspace = (
  { rbacAllPermission, rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    rbacAllPermission ||
    (!!workspace &&
      !!user &&
      (rbacEnabled
        ? permitted.has(V1PermissionType.DELETEWORKSPACE)
        : user.isAdmin || user.id === workspace.userId))
  );
};

const canModifyWorkspace = (
  { rbacAllPermission, rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    rbacAllPermission ||
    (!!workspace &&
      !!user &&
      (rbacEnabled
        ? permitted.has(V1PermissionType.UPDATEWORKSPACE)
        : user.isAdmin || user.id === workspace.userId))
  );
};

const canModifyWorkspaceAgentUserGroup = (
  { rbacAllPermission, rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    rbacAllPermission ||
    (!!user &&
      (rbacEnabled ? permitted.has(V1PermissionType.SETWORKSPACEAGENTUSERGROUP) : user.isAdmin))
  );
};

const canModifyWorkspaceCheckpointStorage = (
  { rbacAllPermission, rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    rbacAllPermission ||
    (!!user &&
      (rbacEnabled
        ? permitted.has(V1PermissionType.SETWORKSPACECHECKPOINTSTORAGECONFIG)
        : user.isAdmin))
  );
};

const canViewWorkspace = (
  { rbacEnabled, rbacReadPermission, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    !!workspace &&
    (!rbacEnabled || rbacReadPermission || permitted.has(V1PermissionType.VIEWWORKSPACE))
  );
};

const canViewWorkspaces = ({
  rbacEnabled,
  rbacReadPermission,
  userRoles,
}: RbacOptsProps): boolean => {
  return (
    !rbacEnabled ||
    rbacReadPermission ||
    (!!userRoles &&
      !!userRoles.find((r) => !!r.permissions.find((p) => p.id === V1PermissionType.VIEWWORKSPACE)))
  );
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
      (rbacEnabled
        ? permitted.has(V1PermissionType.UPDATEROLES)
        : user.isAdmin || user.id === workspace.userId))
  );
};

const canAssignRoles = (
  { rbacAllPermission, rbacEnabled, user, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    rbacAllPermission ||
    (!!user && !!workspace && user.id === workspace.userId) ||
    (!!user && (rbacEnabled ? permitted.has(V1PermissionType.ASSIGNROLES) : user.isAdmin))
  );
};

const canCreateNSC = ({ rbacEnabled, rbacReadPermission, userRoles }: RbacOptsProps): boolean => {
  return (
    !rbacEnabled ||
    rbacReadPermission ||
    (!!userRoles &&
      !!userRoles.find((r) => !!r.permissions.find((p) => p.id === V1PermissionType.CREATENSC)))
  );
};

const canCreateWorkspaceNSC = (
  { rbacAllPermission, rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return (
    rbacAllPermission ||
    (!!workspace && (!rbacEnabled || permitted.has(V1PermissionType.CREATENSC)))
  );
};

const canModifyWorkspaceNSC = (
  { rbacAllPermission, rbacEnabled, userAssignments, userRoles }: RbacOptsProps,
  workspace?: PermissionWorkspace,
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, workspace?.id);
  return rbacAllPermission || !rbacEnabled || permitted.has(V1PermissionType.UPDATENSC);
};

/* Webhooks */

const canEditWebhooks = ({
  rbacAllPermission,
  rbacEnabled,
  user,
  userAssignments,
  userRoles,
}: RbacOptsProps): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles);
  return rbacEnabled
    ? rbacAllPermission || permitted.has(V1PermissionType.EDITWEBHOOKS)
    : !!user && user.isAdmin;
};

export default usePermissions;
