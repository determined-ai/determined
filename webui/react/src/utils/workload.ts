import { RecordKey } from 'types';
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

export const hasCheckpointStep = (step: Type.Step): boolean => {
  return !!step.checkpoint && step.checkpoint.state !== Type.CheckpointState.Deleted;
};

export const workloadsToSteps = (workloads: Type.WorkloadGroup[]): Type.Step[] => {
  const stepsDict: Record<number, Partial<Type.Step>> = {};

  workloads.forEach((workload) => {
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

  return Object.values(stepsDict) as Type.Step[];
};
