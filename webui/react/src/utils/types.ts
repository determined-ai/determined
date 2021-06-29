import { paths } from 'routes/utils';
import { V1ResourcePoolType, V1SchedulerType } from 'services/api-ts-sdk';
import {
  AnyTask, Checkpoint, CheckpointState, CheckpointWorkload, Command, CommandState, CommandTask,
  CommandType, ExperimentItem, Hyperparameters, HyperparameterType, MetricsWorkload,
  RawJson, RecentCommandTask, RecentExperimentTask, RecentTask, RecordKey, ResourceState, RunState,
  SlotState, Workload,
} from 'types';

import { LaunchTensorboardParams } from '../services/types';

import { deletePathList, getPathList, isEqual, isNumber, setPathList } from './data';
import { isMetricsWorkload } from './workload';

/* Conversions to Tasks */

export const commandToTask = (command: CommandTask): RecentCommandTask => {
  return {
    ...command,
    lastEvent: {
      date: command.startTime,
      name: 'requested',
    },
  };
};

export const experimentToTask = (experiment: ExperimentItem): RecentExperimentTask => {
  const lastEvent = experiment.endTime ?
    { date: experiment.endTime, name: 'finished' } :
    { date: experiment.startTime, name: 'requested' };
  const task: RecentTask = {
    archived: experiment.archived,
    id: `${experiment.id}`,
    lastEvent,
    name: experiment.name,
    progress: experiment.progress,
    resourcePool: experiment.resourcePool,
    startTime: experiment.startTime,
    state: experiment.state,
    url: paths.experimentDetails(experiment.id),
    username: experiment.username,
  };
  return task;
};

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
  [CommandType.Notebook]: 'Notebook',
  [CommandType.Shell]: 'Shell',
  [CommandType.Tensorboard]: 'Tensorboard',
};

export function hasKey<O>(obj: O, key: RecordKey): key is keyof O {
  return key in obj;
}

// differentiate Experiment from Task
export const isExperiment = (obj: AnyTask | ExperimentItem): obj is ExperimentItem => {
  return 'config' in obj && 'archived' in obj;
};

// differentiate Experiment from Task
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

/* Experiment Config */
export const trialHParamsToExperimentHParams = (
  hParams: Record<string, unknown>,
): Hyperparameters => {
  const experimentHParams: Hyperparameters = {};
  Object.entries(hParams).forEach(([ param, value ]) => {
    experimentHParams[param] = {
      type: HyperparameterType.Constant,
      val: value as number,
    };
  });
  return experimentHParams;
};

export const getLengthFromStepCount = (config: RawJson, stepCount: number): [string, number] => {
  const DEFAULT_BATCHES_PER_STEP = 100;
  // provide backward compat for step count
  const batchesPerStep = config.batches_per_step || DEFAULT_BATCHES_PER_STEP;
  return [ 'batches', stepCount * batchesPerStep ];
};

const stepRemovalTranslations = [
  { newName: 'searcher.max_length', oldName: 'searcher.max_steps' },
  { oldName: 'min_validation_period' },
  { oldName: 'min_checkpoint_period' },
  { newName: 'searcher.max_length', oldName: 'searcher.target_trial_steps' },
  { newName: 'searcher.length_per_round', oldName: 'searcher.steps_per_round' },
  { newName: 'searcher.budget', oldName: 'searcher.step_budget' },
];

// Add opportunistic backward compatibility to old configs.
export const upgradeConfig = (config: RawJson): void => {
  stepRemovalTranslations.forEach(translation => {
    const oldPath = translation.oldName.split('.');
    const curValue = getPathList<undefined | null | number | unknown>(config, oldPath);
    if (curValue === undefined) return;
    if (curValue === null) {
      deletePathList(config, oldPath);
    }
    if (isNumber(curValue)) {
      const [ key, count ] = getLengthFromStepCount(config, curValue);
      const newPath = (translation.newName || translation.oldName).split('.');
      setPathList(config, newPath, { [key]: count });
      if (translation.newName) deletePathList(config, oldPath);
    }
  });

  delete config.batches_per_step;
};

// Checks whether tensorboard source matches a given source list.
export const tsbMatchesSource =
  (tensorboard: CommandTask, source: LaunchTensorboardParams): boolean => {
    if (source.experimentIds) {
      source.experimentIds?.sort();
      tensorboard.misc?.experimentIds?.sort();

      if (isEqual(tensorboard.misc?.experimentIds, source.experimentIds)) {
        return true;
      }
    }

    if (source.trialIds) {
      source.trialIds?.sort();
      tensorboard.misc?.trialIds?.sort();

      if (isEqual(tensorboard.misc?.trialIds, source.trialIds)) {
        return true;
      }
    }

    return false;
  };

export const getMetricValue = (workload?: Workload, metricName?: string): number | undefined => {
  const metricsWl = workload as MetricsWorkload;
  if (!workload || !isMetricsWorkload(metricsWl) || !metricsWl.metrics) return undefined;

  metricName = metricName || Object.keys(metricsWl.metrics)[0];

  return metricsWl.metrics[metricName];
};

export const getBatchNumber = (
  data: {batch: number} | {totalBatches: number},
): number => {
  if ('batch' in data) {
    return data.batch;
  } else {
    return data.totalBatches;
  }
};

export type Eventually<T> = T | Promise<T>;
