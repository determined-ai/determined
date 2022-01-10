import { updateJobQueue } from 'services/api';
import * as Api from 'services/api-ts-sdk';
import { CommandType, Job, JobState, JobType, ResourcePool } from 'types';
import handleError, { DetError, DetErrorOptions, ErrorType, isDetError } from 'utils/error';

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

export const moveJobToPositionUpdate = (
  jobs: Job[],
  jobId: string,
  position: number,
): Api.V1QueueControl => {
  const errOpts: DetErrorOptions = {
    publicMessage: `Failed to move job to position ${position}`,
    publicSubject: 'TITLE',
    silent: false,
  };
  if (position < 1 || position % 1 !== 0) {
    throw new DetError(
      `Invalid queue position: ${position}.`,
      { ...errOpts, type: ErrorType.Input },
    );
  }
  // what has the same position as the job we want to move?
  const anchorJob = jobs.find(job => job.summary.jobsAhead === position - 1);
  if (!anchorJob) {
    // job view is out of sync.
    // FIXME what's the remedy? They need to retry.
    throw new DetError('Job view is out of sync.', { ...errOpts, type: ErrorType.Ui });
  }

  const isLastJob = jobs.length === position;
  if (isLastJob) {
    return {
      behindOf: anchorJob.jobId,
      jobId,
    };
  }
  return {
    aheadOf: anchorJob.jobId,
    jobId,
  };
};

export const moveJobToPosition = async (
  jobs: Job[],
  jobId: string,
  position: number,
): Promise<void> => {
  try {
    await updateJobQueue(
      { updates: [ moveJobToPositionUpdate(jobs, jobId, position) ] },
    );
  } catch (e) {
    if (isDetError(e)) {
      e.publicMessage = `Failed to move job to position ${position}`;
    }
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
