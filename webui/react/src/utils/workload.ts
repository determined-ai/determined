import { RecordKey } from 'shared/types';
import * as Type from 'types';

// Checkpoint size in bytes.
export const checkpointSize = (checkpoint?: { resources?: Record<RecordKey, number> }): number => {
  if (checkpoint?.resources) {
    return Object.values(checkpoint.resources).reduce((acc, size) => acc + size, 0);
  }
  return 0;
};

export const getWorkload = (
  workload: Type.WorkloadGroup,
): Type.MetricsWorkload | Type.CheckpointWorkload => {
  return Object.values(workload).find((val) => !!val);
};

export const hasCheckpoint = (workload: Type.WorkloadGroup): boolean => {
  return !!workload.checkpoint && workload.checkpoint.state !== Type.CheckpointState.Deleted;
};
