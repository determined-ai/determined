import { IconName } from 'hew/Icon';

import * as Api from 'services/api-ts-sdk';
import { CommandType, Job, JobType, ResourcePool } from 'types';
import { capitalize } from 'utils/string';

export const jobTypeIconName = (jobType: JobType): IconName => {
  if (jobType === JobType.EXTERNAL) {
    return 'external';
  }
  const type = jobTypeToCommandType(jobType);
  return type ?? 'experiment';
};

export const jobTypeLabel = (jobType: JobType): string => {
  return capitalize(jobTypeIconName(jobType));
};

// translate JobType to CommandType
export const jobTypeToCommandType = (jobType: JobType): CommandType | undefined => {
  switch (jobType) {
    case JobType.NOTEBOOK:
      return CommandType.JupyterLab;
    case JobType.SHELL:
      return CommandType.Shell;
    case JobType.TENSORBOARD:
      return CommandType.TensorBoard;
    case JobType.COMMAND:
      return CommandType.Command;
    default:
      return undefined;
  }
};

export const orderedSchedulers = new Set<Api.V1SchedulerType>([
  Api.V1SchedulerType.PRIORITY,
  Api.V1SchedulerType.KUBERNETES,
]);

/*
We cannot modify scheduling parameters of non fault tolerant jobs in Kubernetes.
*/
export const canManageJob = (job: Job, rp?: ResourcePool): boolean => {
  if (!rp) return false;
  return !(rp.schedulerType === Api.V1SchedulerType.KUBERNETES && job.type !== JobType.EXPERIMENT);
};
