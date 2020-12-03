import { CheckpointState, CheckpointWorkload, MetricsWorkload, Step, WorkloadWrapper } from '../types';

export const hasCheckpointStep = (step: Step): boolean => {
  return !!step.checkpoint && step.checkpoint.state !== CheckpointState.Deleted;
};

export const hasCheckpoint = (workload: WorkloadWrapper): boolean => {
  return !!workload.checkpoint && workload.checkpoint.state !== CheckpointState.Deleted;
};

export const getWorkload = (wrapper: WorkloadWrapper): MetricsWorkload | CheckpointWorkload => {
  return Object.values(wrapper).find(val => !!val);
};

export const isMetricsWorkload = (workload: MetricsWorkload | CheckpointWorkload)
: workload is MetricsWorkload => {
  return 'metrics' in workload
  && 'numInputs' in workload
  && !('uuid' in workload)
  && !('resources' in workload);
};
