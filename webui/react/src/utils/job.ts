import * as Api from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { CommandType, Job, JobState, JobType, ResourcePool } from 'types';
import { DetError, ErrorType } from 'utils/error';

import { capitalize } from './string';

export const jobTypeIconName = (jobType: JobType): string => {
  const type = jobTypeToCommandType(jobType);
  if (type) return type.toString();
  return 'experiment';
};

export const jobTypeLabel = (jobType: JobType): string => {
  return capitalize(jobTypeIconName(jobType));
};

// translate JobType to CommandType
export const jobTypeToCommandType = (
  jobType: JobType,
): CommandType | undefined => {
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

export const jobStateToLabel: {[key in JobState]: string} = {
  [JobState.SCHEDULED]: 'Scheduled',
  [JobState.SCHEDULEDBACKFILLED]: 'ScheduledBackfilled',
  [JobState.QUEUED]: 'Queued',
};

export const orderedSchedulers = new Set(
  [ Api.V1SchedulerType.PRIORITY, Api.V1SchedulerType.KUBERNETES ],
);

export const moveJobToPositionUpdate = (jobId: string, position: number): Api.V1QueueControl => {
  if (position < 1 || position % 1 !== 0) {
    throw new DetError(`Invalid queue position: ${position}.`, { type: ErrorType.Input });
  }
  return {
    jobId,
    queuePosition: position - 1,
  };
};

export const moveJobToPosition = async (jobId: string, position: number): Promise<void> => {
  await detApi.Internal.updateJobQueue({ updates: [ moveJobToPositionUpdate(jobId, position) ] });
};

/*
We cannot modify scheduling parameters of non fault tolerant jobs in Kubernetes.
*/
export const canManageJob = (job: Job, rp?: ResourcePool): boolean => {
  if (!rp) return false;
  return !(rp.schedulerType === Api.V1SchedulerType.KUBERNETES &&
    job.type !== JobType.EXPERIMENT);
};
