import { updateJobQueue } from 'services/api';
import * as Api from 'services/api-ts-sdk';
import { DetError, DetErrorOptions, ErrorType, wrapPublicMessage } from 'shared/utils/error';
import { capitalize } from 'shared/utils/string';
import { CommandType, Job, JobType, ResourcePool } from 'types';
import handleError from 'utils/error';

// This marks scheduler types that do not support fine-grain control of
// job positions in the queue.
export const unsupportedQPosSchedulers = new Set([ Api.V1SchedulerType.FAIRSHARE ]);

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

export const orderedSchedulers = new Set(
  [ Api.V1SchedulerType.PRIORITY, Api.V1SchedulerType.KUBERNETES ],
);

export const moveJobToTop = async (
  curTopJob: Job,
  targetJob: Job,
): Promise<void> => {
  if (curTopJob.jobId === targetJob.jobId || targetJob.summary.jobsAhead === 1) {
    return; // no op
  }
  try {
    const update = {
      aheadOf: curTopJob.jobId,
      jobId: targetJob.jobId,
    };
    await updateJobQueue({ updates: [ update ] });
  } catch (e) {
    handleError(e, { publicMessage: wrapPublicMessage(e, 'Failed to move job to top') });
  }
};

/*
We cannot modify scheduling parameters of non fault tolerant jobs in Kubernetes.
*/
export const canManageJob = (job: Job, rp?: ResourcePool): boolean => {
  if (!rp) return false;
  return !(rp.schedulerType === Api.V1SchedulerType.KUBERNETES &&
    job.type !== JobType.EXPERIMENT);
};
