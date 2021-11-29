import { V1ResourcePoolType, V1SchedulerType } from 'services/api-ts-sdk';
import {
  AnyTask, Checkpoint, CheckpointState, CheckpointWorkload, Command, CommandState,
  CommandTask, CommandType, ExperimentItem, RecordKey, ResourceState, RunState, SlotState,
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

export const killableRunStates = [ RunState.Active, RunState.Paused, RunState.StoppingCanceled ];
export const cancellableRunStates = [ RunState.Active, RunState.Paused ];
export const killableCmdStates = [
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

export const terminalRunStates: Set<RunState> = new Set([
  RunState.Canceled,
  RunState.Completed,
  RunState.Errored,
  RunState.Deleted,
]);

export const deletableRunStates: Set<RunState> = new Set([
  RunState.Canceled,
  RunState.Completed,
  RunState.Errored,
  RunState.DeleteFailed,
]);

export const runStateToLabel: {[key in RunState]: string} = {
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

export const V1ResourcePoolTypeToLabel: {[key in V1ResourcePoolType]: string} = {
  [V1ResourcePoolType.UNSPECIFIED]: 'Unspecified',
  [V1ResourcePoolType.AWS]: 'AWS',
  [V1ResourcePoolType.GCP]: 'GCP',
  [V1ResourcePoolType.STATIC]: 'Static',
  [V1ResourcePoolType.K8S]: 'Kubernetes',
};

export const V1SchedulerTypeToLabel : {[key in V1SchedulerType]: string} = {
  [V1SchedulerType.FAIRSHARE]: 'Fairshare',
  [V1SchedulerType.KUBERNETES]: 'Kubernetes',
  [V1SchedulerType.PRIORITY]: 'Priority',
  [V1SchedulerType.ROUNDROBIN]: 'RoundRobin',
  [V1SchedulerType.UNSPECIFIED]: 'Unspecified',
};

export const commandStateToLabel: {[key in CommandState]: string} = {
  [CommandState.Pending]: 'Pending',
  [CommandState.Assigned]: 'Assigned',
  [CommandState.Pulling]: 'Pulling',
  [CommandState.Starting]: 'Starting',
  [CommandState.Running]: 'Running',
  [CommandState.Terminating]: 'Terminating',
  [CommandState.Terminated]: 'Terminated',
};

export const slotStateToLabel: {[key in SlotState]: string} = {
  [SlotState.Pending]: 'Pending',
  [SlotState.Running]: 'Running',
  [SlotState.Free]: 'Free',
};

export const checkpointStateToLabel: {[key in CheckpointState]: string} = {
  [CheckpointState.Active]: 'Active',
  [CheckpointState.Completed]: 'Completed',
  [CheckpointState.Error]: 'Error',
  [CheckpointState.Deleted]: 'Deleted',
  [CheckpointState.Unspecified]: 'Unspecified',
};

export const resourceStateToLabel: {[key in ResourceState]: string} = {
  [ResourceState.Running]: 'Running',
  [ResourceState.Assigned]: 'Assigned',
  [ResourceState.Terminated]: 'Terminated',
  [ResourceState.Pulling]: 'Pulling',
  [ResourceState.Starting]: 'Starting',
  [ResourceState.Unspecified]: 'Unspecified',
};

export const isTaskKillable = (task: AnyTask | ExperimentItem): boolean => {
  return killableRunStates.includes(task.state as RunState)
    || killableCmdStates.includes(task.state as CommandState);
};

export function stateToLabel(
  state: RunState | CommandState | CheckpointState | ResourceState | SlotState,
): string {
  return runStateToLabel[state as RunState]
  || commandStateToLabel[state as CommandState]
  || resourceStateToLabel[state as ResourceState]
  || checkpointStateToLabel[state as CheckpointState]
  || slotStateToLabel[state as SlotState];
}

export const commandTypeToLabel: {[key in CommandType]: string} = {
  [CommandType.Command]: 'Command',
  [CommandType.JupyterLab]: 'JupyterLab',
  [CommandType.Shell]: 'Shell',
  [CommandType.TensorBoard]: 'TensorBoard',
};

export function hasKey<O>(obj: O, key: RecordKey): key is keyof O {
  return key in obj;
}

// differentiate Experiment from Task
export const isExperiment = (obj: AnyTask | ExperimentItem): obj is ExperimentItem => {
  return 'config' in obj && 'archived' in obj;
};

// differentiate Task from Experiment
export const isCommandTask = (obj: Command | CommandTask): obj is CommandTask => {
  return 'type' in obj;
};

// used when properties are named differently between objects.
/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const oneOfProperties = <T>(obj: any, props: string[]): T => {
  for (const prop of props) {
    if (prop in obj) return obj[prop] as T;
  }
  throw new Error('no matching property');
};

// size in bytes
export const checkpointSize = (checkpoint: Checkpoint | CheckpointWorkload): number => {
  if (!checkpoint.resources) return 0;
  const total = Object.values(checkpoint.resources).reduce((acc, size) => acc + size, 0);
  return total;
};

export const getBatchNumber = (
  data: { batch: number } | { totalBatches: number },
): number => {
  if ('batch' in data) {
    return data.batch;
  } else {
    return data.totalBatches;
  }
};

export type Eventually<T> = T | Promise<T>;
