import { RecordKey } from 'shared/types';
import * as Type from 'types';

// Checkpoint size in bytes.
export const checkpointSize = (
  checkpoint?: { resources?: Record<RecordKey, number> },
): number => {
  if (checkpoint?.resources) {
    return Object.values(checkpoint.resources).reduce((acc, size) => acc + size, 0);
  }
  return 0;
};

export const hasCheckpoint = (workload: Type.WorkloadGroup): boolean => {
  return !!workload.checkpoint && workload.checkpoint.state !== Type.CheckpointState.Deleted;
};

export const hasCheckpointStep = (step: Type.Step): boolean => {
  return !!step.checkpoint && step.checkpoint.state !== Type.CheckpointState.Deleted;
};

export const isMetricsWorkload = (
  workload: Type.MetricsWorkload | Type.CheckpointWorkload,
): workload is Type.MetricsWorkload => {
  if ('uuid' in workload || 'resources' in workload) return false;
  if ('metrics' in workload) return true;
  // we can't determine which one it is.
  return false;
};

export const workloadsToSteps = (workloads: Type.WorkloadGroup[]): Type.Step[] => {
  return workloads.map(workload => {
    let wltype = 't';
    if (workload.validation) {
      wltype = 'v';
    } else if (workload.checkpoint) {
      wltype = 'c';
    }
    const batchNum = (
      workload?.checkpoint || workload?.validation || workload?.training
    )?.totalBatches;
    return {
      batchNum: batchNum,
      checkpoint: workload.checkpoint,
      key: wltype + batchNum,
      startTime: '',
      training: workload.training,
      validation: workload.validation,
    } as Type.Step;
  });
};
