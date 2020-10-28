import { CheckpointState, Step } from '../types';

export const hasCheckpoint = (step: Step): boolean => {
  return !!step.checkpoint && step.checkpoint.state !== CheckpointState.Deleted;
};
