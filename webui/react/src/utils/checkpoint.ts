import {
  CheckpointAction,
  checkpointAction,
  CheckpointState,
  CoreApiGenericCheckpoint,
} from 'types';

type CheckpointChecker = (checkpoint: CoreApiGenericCheckpoint) => boolean;

/* eslint-disable @typescript-eslint/no-unused-vars */
export const alwaysTrueCheckpointChecker = (checkpoint: CoreApiGenericCheckpoint): boolean => true;

const CheckpointCheckers: Record<CheckpointAction, CheckpointChecker> = {
  /**
   * for internal use: the typing ensures that checkers
   * are defined for every CheckpointAction
   * we expose the functions below as convenient wrappers
   */

  [checkpointAction.Delete]: alwaysTrueCheckpointChecker,

  [checkpointAction.Register]: (checkpoint) => checkpoint.state === CheckpointState.Completed,
};

export const canActionCheckpoint = (
  action: CheckpointAction,
  checkpoint: CoreApiGenericCheckpoint,
): boolean => !!checkpoint && CheckpointCheckers[action](checkpoint);

export const getActionsForCheckpoint = (
  checkpoint: CoreApiGenericCheckpoint,
  targets: CheckpointAction[],
): CheckpointAction[] => {
  if (!checkpoint) return []; // redundant, for clarity
  return targets.filter((action) => canActionCheckpoint(action, checkpoint));
};

export const getActionsForCheckpointsUnion = (
  checkpoints: CoreApiGenericCheckpoint[],
  targets: CheckpointAction[],
): CheckpointAction[] => {
  if (!checkpoints.length) return []; // redundant, for clarity
  const actionsForCheckpoints = checkpoints.map((e) => getActionsForCheckpoint(e, targets));
  return targets.filter((action) =>
    actionsForCheckpoints.some((checkpointActions) => checkpointActions.includes(action)),
  );
};

export const getActionsForCheckpointsIntersection = (
  checkpoints: CoreApiGenericCheckpoint[],
  targets: CheckpointAction[],
): CheckpointAction[] => {
  if (!checkpoints.length) [];
  const actionsForCheckpoints = checkpoints.map((e) => getActionsForCheckpoint(e, targets));
  return targets.filter((action) =>
    actionsForCheckpoints.every((checkpointActions) => checkpointActions.includes(action)),
  );
};
