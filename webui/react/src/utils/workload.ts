import {
  CheckpointState, CheckpointWorkload, MetricsWorkload, Step, WorkloadGroup,
} from '../types';

export const getWorkload = (workload: WorkloadGroup): MetricsWorkload | CheckpointWorkload => {
  return Object.values(workload).find(val => !!val);
};

export const hasCheckpointStep = (step: Step): boolean => {
  return !!step.checkpoint && step.checkpoint.state !== CheckpointState.Deleted;
};

export const hasCheckpoint = (workload: WorkloadGroup): boolean => {
  return !!workload.checkpoint && workload.checkpoint.state !== CheckpointState.Deleted;
};

export const isMetricsWorkload = (
  workload: MetricsWorkload | CheckpointWorkload,
): workload is MetricsWorkload => {
  if ('uuid' in workload || 'resources' in workload) return false;
  if ('metrics' in workload) return true;
  // we can't determine which one it is.
  return false;
};

export const workloadsToSteps = (workloads: WorkloadGroup[]): Step[] => {
  const stepsDict: Record<number, Partial<Step>> = {};

  workloads.forEach(workload => {
    const wl = getWorkload(workload);
    const batchNum = wl.totalBatches;
    if (stepsDict[batchNum] === undefined) stepsDict[batchNum] = {};
    stepsDict[batchNum].batchNum = batchNum;

    if (workload.checkpoint) {
      stepsDict[batchNum].checkpoint = workload.checkpoint;
    } else if (workload.validation) {
      stepsDict[batchNum].validation = workload.validation;
    } else if (workload.training) {
      stepsDict[batchNum].training = workload.training;
    }
  });

  return Object.values(stepsDict) as Step[];
};
