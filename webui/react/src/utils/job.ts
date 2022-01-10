import { AnyTask, CommandState, CommandTask, CommandType, ExperimentTask, Job, JobState, JobType,
  RunState } from 'types';

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

export const taskFromJob = (job: Job): AnyTask => {
  const baseTask = { id: job.entityId, name: job.name };
  let rv: AnyTask;
  if (job.type === JobType.EXPERIMENT) {
    rv = { ...baseTask, archived: false, state: RunState.Active } as ExperimentTask;
  } else {
    rv = {
      ...baseTask,
      state: CommandState.Running,
      type: jobTypeToCommandType(job.type),
    } as CommandTask;
  }
  return rv;
};

export const jobStateToLabel: {[key in JobState]: string} = {
  [JobState.SCHEDULED]: 'Scheduled',
  [JobState.SCHEDULEDBACKFILLED]: 'ScheduledBackfilled',
  [JobState.QUEUED]: 'Queued',
};
