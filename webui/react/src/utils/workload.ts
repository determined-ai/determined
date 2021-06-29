import { CheckpointWorkload, MetricsWorkload, Step,
  WorkloadWrapper } from '../types';

export const hasCheckpointStep = (step: Step): boolean => {
  return !!step.checkpoint;
};

export const hasCheckpoint = (workload: WorkloadWrapper): boolean => {
  return !!workload.checkpoint;
};

export const getWorkload = (wrapper: WorkloadWrapper): MetricsWorkload | CheckpointWorkload => {
  return Object.values(wrapper).find(val => !!val);
};

export const isMetricsWorkload = (workload: MetricsWorkload | CheckpointWorkload)
: workload is MetricsWorkload => {
  if ('uuid' in workload || 'resources' in workload) return false;
  if ('metrics' in workload || 'numInputs' in workload) return true;
  // we can't determine which one it is.
  return false;
};

export const workloadsToSteps = (workloads: WorkloadWrapper[]): Step[] => {
  const stepsDict: Record<number, Partial<Step>> = {};
  workloads.forEach(wlWrapper => {
    const wl = getWorkload(wlWrapper);
    const batchNum = wl.totalBatches;
    if (stepsDict[batchNum] === undefined) stepsDict[batchNum] = {};
    stepsDict[batchNum].batchNum = batchNum;

    if (wlWrapper.checkpoint) {
      stepsDict[batchNum].checkpoint = wlWrapper.checkpoint;
    } else if (wlWrapper.validation) {
      stepsDict[batchNum].validation = wlWrapper.validation;
    } else if (wlWrapper.training) {
      stepsDict[batchNum].training = wlWrapper.training;
    }
  });
  return Object.values(stepsDict) as Step[];
};
