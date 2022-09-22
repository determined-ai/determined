import { V1ResourcePoolType, V1SchedulerType } from 'services/api-ts-sdk';
import { StateOfUnion } from 'themes';
import {
  CheckpointState,
  CommandState,
  CommandType,
  CompoundRunState,
  JobState,
  ResourceState,
  RunState,
  RunStateValue,
  SlotState,
} from 'types';

// TODO: probably its a good idea to add library like underscore or similar one
const pick = <T, K extends keyof T>(obj: T, ...keys: K[]): Pick<T, K> => {
  const ret: Pick<T, K> = {} as Pick<T, K>;
  keys.forEach((key) => {
    ret[key] = obj[key];
  });
  return ret;
};

const keysOf = <T>(obj: T): Array<keyof T> => {
  const ret = [];
  for (const key in obj) {
    ret.push(key);
  }
  return ret;
};

export const activeCommandStates = [
  CommandState.Pulling,
  CommandState.Queued,
  CommandState.Running,
  CommandState.Starting,
  CommandState.Terminating,
];

export const activeRunStates: Array<
  'STATE_ACTIVE' | 'STATE_STOPPING_COMPLETED' | 'STATE_STOPPING_CANCELED' | 'STATE_STOPPING_ERROR'
> = ['STATE_ACTIVE', 'STATE_STOPPING_CANCELED', 'STATE_STOPPING_COMPLETED', 'STATE_STOPPING_ERROR'];

/* activeStates are sub-states which replace the previous Active RunState,
  and Active for backward compatibility  */
const activeStates: Array<RunStateValue> = [
  RunState.ACTIVE,
  RunState.PULLING,
  RunState.QUEUED,
  RunState.RUNNING,
  RunState.STARTING,
];
const jobStates: Array<JobState> = [
  JobState.QUEUED,
  JobState.SCHEDULED,
  JobState.SCHEDULEDBACKFILLED,
];
export const killableRunStates: CompoundRunState[] = [
  ...activeStates,
  RunState.PAUSED,
  RunState.STOPPING_CANCELED,
  ...jobStates,
];

export const pausableRunStates: Set<CompoundRunState> = new Set([...activeStates, ...jobStates]);

export const cancellableRunStates: Set<CompoundRunState> = new Set([
  ...activeStates,
  RunState.PAUSED,
  ...jobStates,
]);

export const killableCommandStates = [
  CommandState.Pulling,
  CommandState.Queued,
  CommandState.Running,
  CommandState.Starting,
];

export const terminalCommandStates: Set<CommandState> = new Set([
  CommandState.Terminated,
  CommandState.Terminating,
]);

const runStateList = [
  RunState.CANCELED,
  RunState.COMPLETED,
  RunState.ERROR,
  RunState.DELETE_FAILED,
] as const;

export const deletableRunStates: Set<CompoundRunState> = new Set(runStateList);

export const terminalRunStates: Set<CompoundRunState> = new Set([
  ...deletableRunStates,
  RunState.DELETED,
]);

export const terminalRunStatesKeys = keysOf({
  ...pick(RunState, ...runStateList),
  ...pick(RunState, RunState.DELETED),
});

export const runStateToLabel: { [key in RunStateValue]: string } = {
  [RunState.ACTIVE]: 'Active',
  [RunState.RUNNING]: 'Running',
  [RunState.CANCELED]: 'Canceled',
  [RunState.COMPLETED]: 'Completed',
  [RunState.DELETED]: 'Deleted',
  [RunState.DELETING]: 'Deleting',
  [RunState.DELETE_FAILED]: 'Delete Failed',
  [RunState.ERROR]: 'Errored',
  [RunState.PAUSED]: 'Paused',
  [RunState.STOPPING_CANCELED]: 'Canceling',
  [RunState.STOPPING_COMPLETED]: 'Completing',
  [RunState.STOPPING_ERROR]: 'Erroring',
  [RunState.STOPPING_KILLED]: 'Killed',
  [RunState.UNSPECIFIED]: 'Unspecified',
  [RunState.QUEUED]: 'Queued',
  [RunState.PULLING]: 'Pulling Image',
  [RunState.STARTING]: 'Running (preparing env)',
};

export const V1ResourcePoolTypeToLabel: { [key in V1ResourcePoolType]: string } = {
  [V1ResourcePoolType.UNSPECIFIED]: 'Unspecified',
  [V1ResourcePoolType.AWS]: 'AWS',
  [V1ResourcePoolType.GCP]: 'GCP',
  [V1ResourcePoolType.STATIC]: 'Static',
  [V1ResourcePoolType.K8S]: 'Kubernetes',
};

export const V1SchedulerTypeToLabel: { [key in V1SchedulerType]: string } = {
  [V1SchedulerType.FAIRSHARE]: 'Fairshare',
  [V1SchedulerType.KUBERNETES]: 'Kubernetes',
  [V1SchedulerType.PRIORITY]: 'Priority',
  [V1SchedulerType.ROUNDROBIN]: 'RoundRobin',
  [V1SchedulerType.SLURM]: 'Slurm',
  [V1SchedulerType.PBS]: 'PBS',
  [V1SchedulerType.UNSPECIFIED]: 'Unspecified',
};

export const commandStateToLabel: { [key in CommandState]: string } = {
  [CommandState.Waiting]: 'Waiting',
  [CommandState.Pulling]: 'Pulling',
  [CommandState.Queued]: 'Queued',
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

export function stateToLabel(state: StateOfUnion): string {
  return (
    runStateToLabel[state as RunStateValue] ||
    commandStateToLabel[state as CommandState] ||
    resourceStateToLabel[state as ResourceState] ||
    checkpointStateToLabel[state as CheckpointState] ||
    jobStateToLabel[state as JobState] ||
    slotStateToLabel[state as SlotState] ||
    (state as string)
  );
}
