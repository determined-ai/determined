import { MetricsWorkload, RecordKey } from 'types';
import * as Type from 'types';

// Checkpoint size in bytes.
export const checkpointSize = (checkpoint?: { resources?: Record<RecordKey, number> }): number => {
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

export const workloadsToSteps = (workloads: Type.WorkloadGroup[]): Type.Step[] => {
  const stepsDict: Record<number, Partial<Type.Step>> = {};

  workloads.forEach((workload) => {
    let batchNum: number | undefined = undefined;
    if (workload.checkpoint) {
      batchNum = workload.checkpoint.totalBatches;
      stepsDict[batchNum] = stepsDict[batchNum] ?? {};
      stepsDict[batchNum].batchNum = batchNum;
      stepsDict[batchNum].checkpoint = workload.checkpoint;
    } else {
      for (const group in workload.metrics) {
        batchNum = workload.metrics[group].totalBatches;
        stepsDict[batchNum] = stepsDict[batchNum] ?? {};
        stepsDict[batchNum].batchNum = batchNum;
        stepsDict[batchNum].metrics = stepsDict[batchNum].metrics ?? {};
        (stepsDict[batchNum].metrics as Record<string, MetricsWorkload>)[group] =
          workload.metrics[group];
      }
    }
  });

  return Object.values(stepsDict) as Type.Step[];
};
