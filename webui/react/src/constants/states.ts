import { V1ResourcePoolType, V1SchedulerType } from 'services/api-ts-sdk';
import { StateOfUnion } from 'themes';
import {
  CheckpointState, CommandState, CommandType, CompoundRunState, JobState, ResourceState, RunState,
  SlotState,
} from 'types';

export const activeCommandStates = [
  CommandState.Assigned,
  CommandState.Pending,
  CommandState.Pulling,
  CommandState.Running,
  CommandState.Starting,
  CommandState.Terminating,
];

export const activeRunStates: Array<
  'STATE_ACTIVE' | 'STATE_STOPPING_COMPLETED' | 'STATE_STOPPING_CANCELED' | 'STATE_STOPPING_ERROR'
> = [
  'STATE_ACTIVE',
  'STATE_STOPPING_CANCELED',
  'STATE_STOPPING_COMPLETED',
  'STATE_STOPPING_ERROR',
];

const jobStates: Array<JobState> = [
  JobState.QUEUED, JobState.SCHEDULED, JobState.SCHEDULEDBACKFILLED,
];
export const killableRunStates: CompoundRunState[] =
  [ RunState.Active,
    RunState.Paused,
    RunState.StoppingCanceled,
    ...jobStates,
  ];

export const pausableRunStates: Set<CompoundRunState> = new Set([ RunState.Active, ...jobStates ]);

export const cancellableRunStates: Set<CompoundRunState> = new Set([
  RunState.Active,
  RunState.Paused,
  ...jobStates,
]);

export const killableCommandStates = [
  CommandState.Assigned,
  CommandState.Pending,
  CommandState.Pulling,
  CommandState.Running,
  CommandState.Starting,
];

export const terminalCommandStates: Set<CommandState> = new Set([
  CommandState.Terminated,
  CommandState.Terminating,
]);

export const deletableRunStates: Set<CompoundRunState> = new Set([
  RunState.Canceled,
  RunState.Completed,
  RunState.Errored,
  RunState.DeleteFailed,
]);

export const terminalRunStates: Set<CompoundRunState> = new Set([
  ...deletableRunStates,
  RunState.Deleted,
]);

export const runStateToLabel: { [key in RunState]: string } = {
  [RunState.Active]: 'Active',
  [RunState.Canceled]: 'Canceled',
  [RunState.Completed]: 'Completed',
  [RunState.Deleted]: 'Deleted',
  [RunState.Deleting]: 'Deleting',
  [RunState.DeleteFailed]: 'Delete Failed',
  [RunState.Errored]: 'Errored',
  [RunState.Paused]: 'Paused',
  [RunState.StoppingCanceled]: 'Canceling',
  [RunState.StoppingCompleted]: 'Completing',
  [RunState.StoppingError]: 'Erroring',
  [RunState.Unspecified]: 'Unspecified',
};

export const V1ResourcePoolTypeToLabel: { [key in V1ResourcePoolType]: string } = {
  [V1ResourcePoolType.UNSPECIFIED]: 'Unspecified',
  [V1ResourcePoolType.AWS]: 'AWS',
  [V1ResourcePoolType.GCP]: 'GCP',
  [V1ResourcePoolType.STATIC]: 'Static',
  [V1ResourcePoolType.K8S]: 'Kubernetes',
};

export const V1SchedulerTypeToLabel : { [key in V1SchedulerType]: string } = {
  [V1SchedulerType.FAIRSHARE]: 'Fairshare',
  [V1SchedulerType.KUBERNETES]: 'Kubernetes',
  [V1SchedulerType.PRIORITY]: 'Priority',
  [V1SchedulerType.ROUNDROBIN]: 'RoundRobin',
  [V1SchedulerType.SLURM]: 'Slurm',
  [V1SchedulerType.PBS]: 'PBS',
  [V1SchedulerType.UNSPECIFIED]: 'Unspecified',
};

export const commandStateToLabel: { [key in CommandState]: string } = {
  [CommandState.Pending]: 'Pending',
  [CommandState.Assigned]: 'Assigned',
  [CommandState.Pulling]: 'Pulling',
  [CommandState.Starting]: 'Starting',
  [CommandState.Running]: 'Running',
  [CommandState.Terminating]: 'Terminating',
  [CommandState.Terminated]: 'Terminated',
};

export const checkpointStateToLabel: { [key in CheckpointState]: string } = {
  [CheckpointState.Active]: 'Active',
  [CheckpointState.Completed]: 'Completed',
  [CheckpointState.Error]: 'Error',
  [CheckpointState.Deleted]: 'Deleted',
  [CheckpointState.Unspecified]: 'Unspecified',
};

export const resourceStateToLabel: { [key in ResourceState]: string } = {
  [ResourceState.Running]: 'Running',
  [ResourceState.Assigned]: 'Assigned',
  [ResourceState.Terminated]: 'Terminated',
  [ResourceState.Pulling]: 'Pulling',
  [ResourceState.Starting]: 'Starting',
  [ResourceState.Warm]: 'Warm',
  [ResourceState.Potential]: 'Potential',
  [ResourceState.Unspecified]: 'Unspecified',
};

export const commandTypeToLabel: { [key in CommandType]: string } = {
  [CommandType.Command]: 'Command',
  [CommandType.JupyterLab]: 'JupyterLab',
  [CommandType.Shell]: 'Shell',
  [CommandType.TensorBoard]: 'TensorBoard',
};

export const jobStateToLabel: { [key in JobState]: string } = {
  [JobState.SCHEDULED]: 'Scheduled',
  [JobState.SCHEDULEDBACKFILLED]: 'ScheduledBackfilled',
  [JobState.QUEUED]: 'Queued',
};

export const slotStateToLabel: { [key in SlotState]: string } = {
  [SlotState.Pending]: 'Pending',
  [SlotState.Running]: 'Running',
  [SlotState.Free]: 'Free',
  [SlotState.Potential]: 'Potential',
};

export function stateToLabel(
  state: StateOfUnion,
): string {
  return runStateToLabel[state as RunState]
    || commandStateToLabel[state as CommandState]
    || resourceStateToLabel[state as ResourceState]
    || checkpointStateToLabel[state as CheckpointState]
    || jobStateToLabel[state as JobState]
    || slotStateToLabel[state as SlotState]
    || state as string;
}
