import { updateJobQueue } from 'services/api';
import * as Api from 'services/api-ts-sdk';
import { CommandType, Job, JobType, ResourcePool } from 'types';
import handleError, { DetError, DetErrorOptions, ErrorType } from 'utils/error';

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

export const orderedSchedulers = new Set(
  [ Api.V1SchedulerType.PRIORITY, Api.V1SchedulerType.KUBERNETES ],
);

/**
 * Create the update request based on a given position for a job.
 * @throws {DetError}
 */
export const moveJobToPositionUpdate = (
  jobs: Job[],
  jobId: string,
  position: number,
): Api.V1QueueControl | undefined => {
  const errOpts: DetErrorOptions = {
    isUserTriggered: true,
    publicMessage: `Failed to move job to position ${position}.`,
    publicSubject: 'Moving job failed.',
    silent: false,
  };
  if (position < 1 || position % 1 !== 0) {
    throw new DetError(
      `Invalid queue position: ${position}.`,
      { ...errOpts, type: ErrorType.Input },
    );
  }
  const anchorJob = jobs.find(job => job.summary.jobsAhead === position - 1);
  const job = jobs.find(job => job.jobId === jobId);

  if (!anchorJob || !job) {
    // job view is out of sync.
    throw new DetError('Job view is out of sync.', {
      ...errOpts,
      publicMessage: 'Please retry.',
      type: ErrorType.Ui,
    });
  }

  if (anchorJob.jobId === jobId || job.summary.jobsAhead === position - 1) {
    return; // no op
  }

  const isMovingAhead = job.summary.jobsAhead >= position;
  if (isMovingAhead) {
    return {
      aheadOf: anchorJob.jobId,
      jobId,
    };
  } else {
    return {
      behindOf: anchorJob.jobId,
      jobId,
    };
  }
};

export const moveJobToPosition = async (
  jobs: Job[],
  jobId: string,
  position: number,
): Promise<void> => {
  try {
    const update = moveJobToPositionUpdate(jobs, jobId, position);
    if (update) await updateJobQueue({ updates: [ update ] });
  } catch (e) {
    handleError(e);
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
