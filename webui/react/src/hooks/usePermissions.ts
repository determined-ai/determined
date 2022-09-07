import { useStore } from 'contexts/Store';
import { ExperimentPermissionsArgs, ModelItem, ModelVersion, Project } from 'types';
import {
  canDeleteExperiment,
  canDeleteModel,
  canDeleteModelVersion,
  canDeleteWorkspace,
  canDeleteWorkspaceProjects,
  canModifyWorkspace,
  canModifyWorkspaceProjects,
  canMoveExperiment,
  canMoveWorkspaceProjects,
  PermissionWorkspace,
} from 'utils/role';

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

interface WorkspacePermissionsArgs {
  workspace?: PermissionWorkspace;
}

interface PermissionsHook {
  canDeleteExperiment: (arg0: ExperimentPermissionsArgs) => boolean;
  canDeleteModel: (arg0: ModelPermissionsArgs) => boolean;
  canDeleteModelVersion: (arg0: ModelVersionPermissionsArgs) => boolean;
  canDeleteProjects: (arg0: ProjectPermissionsArgs) => boolean;
  canDeleteWorkspace: (arg0: WorkspacePermissionsArgs) => boolean;
  canModifyProjects: (arg0: ProjectPermissionsArgs) => boolean;
  canModifyWorkspace: (arg0: WorkspacePermissionsArgs) => boolean;
  canMoveExperiment: (arg0: ExperimentPermissionsArgs) => boolean;
  canMoveProjects: (arg0: ProjectPermissionsArgs) => boolean;
}

const usePermissions = (): PermissionsHook => {
  const { auth: { user }, userAssignments, userRoles } = useStore();

  return {
    canDeleteExperiment: (args: ExperimentPermissionsArgs) => canDeleteExperiment(
      args.experiment,
      user,
      userAssignments,
      userRoles,
    ),
    canDeleteModel: (args: ModelPermissionsArgs) => canDeleteModel(
      args.model,
      user,
      userAssignments,
      userRoles,
    ),
    canDeleteModelVersion: (args: ModelVersionPermissionsArgs) => canDeleteModelVersion(
      args.modelVersion,
      user,
      userAssignments,
      userRoles,
    ),
    canDeleteProjects: (args: ProjectPermissionsArgs) => canDeleteWorkspaceProjects(
      args.workspace,
      args.project,
      user,
      userAssignments,
      userRoles,
    ),
    canDeleteWorkspace: (args: WorkspacePermissionsArgs) => canDeleteWorkspace(
      args.workspace,
      user,
      userAssignments,
      userRoles,
    ),
    canModifyProjects: (args: ProjectPermissionsArgs) => canModifyWorkspaceProjects(
      args.workspace,
      args.project,
      user,
      userAssignments,
      userRoles,
    ),
    canModifyWorkspace: (args: WorkspacePermissionsArgs) => canModifyWorkspace(
      args.workspace,
      user,
      userAssignments,
      userRoles,
    ),
    canMoveExperiment: (args: ExperimentPermissionsArgs) => canMoveExperiment(
      args.experiment,
      user,
      userAssignments,
      userRoles,
    ),
    canMoveProjects: (args: ProjectPermissionsArgs) => canMoveWorkspaceProjects(
      args.workspace,
      args.project,
      user,
      userAssignments,
      userRoles,
    ),
  };
};

export default usePermissions;
