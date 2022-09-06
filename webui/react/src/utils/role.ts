import {
  DetailedUser,
  ModelItem,
  ModelVersion,
  Permission,
  ProjectExperiment,
  UserAssignment,
  UserRole,
} from 'types';

// Permissions inside this workspace scope (no workspace = cluster-wide scope)
const relevantPermissions = (
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[],
  workspaceId?: number,
): Set<string> => {
  if (!userAssignments || !userRoles) {
    return new Set<string>();
  }
  const relevantAssigned = userAssignments.filter((a) => a.cluster ||
    (workspaceId && a.workspaces && a.workspaces.includes(workspaceId))).map((a) => a.name);
  let permissions = Array<Permission>();
  userRoles.filter((r) => relevantAssigned.includes(r.name)).forEach((r) => {
    // TODO: is it possible a role is assigned to this workspace,
    // but not all of its permissions?
    permissions = permissions.concat(r.permissions.filter((p) => p.globalOnly || workspaceId));
  });
  return new Set<string>(permissions.map((p) => p.name));
};

// Experiment actions
export const canDeleteExperiment = (
  experiment: ProjectExperiment,
  user: DetailedUser,
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[],
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles, experiment.workspaceId);
  return !!experiment && !!user &&
    permitted.has('oss_user') ? (user.isAdmin || user.id === experiment.userId)
    : permitted.has('delete_experiment');
};

// Model and ModelVersion actions
export const canDeleteModel = (
  model: ModelItem,
  userId?: number,
  userAdmin?: boolean,
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[],
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles);
  return !!model && !!userId &&
    permitted.has('oss_user') ? (userAdmin || userId === model.userId)
    : permitted.has('delete_model');
};

export const canDeleteModelVersion = (
  modelVersion: ModelVersion | undefined,
  user: DetailedUser | undefined,
  userAssignments?: UserAssignment[],
  userRoles?: UserRole[],
): boolean => {
  const permitted = relevantPermissions(userAssignments, userRoles);
  return !!modelVersion && !!user &&
    permitted.has('oss_user') ? (user.isAdmin || user.id === modelVersion.userId)
    : permitted.has('delete_model_version');
};

// Project actions

// Workspace actions
