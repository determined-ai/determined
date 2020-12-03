import {
  AnyTask, Checkpoint, CheckpointState, CheckpointWorkload, Command, CommandState, CommandTask, CommandType, ExperimentHyperParams,
  ExperimentItem, RawJson, RecentCommandTask, RecentExperimentTask, RecentTask, RunState, Step,
  TBSource, TBSourceType, Workload, WorkloadWrapper,
} from 'types';

import { deletePathList, getPathList, isEqual, isNumber, setPathList } from './data';
import { getDuration } from './time';

/* Conversions to Tasks */

export const commandToTask = (command: Command): RecentCommandTask => {
  // We expect the name to be in the form of 'Type (pet-name-generated)'.
  const name = command.config.description.replace(/.*\((.*)\).*/, '$1');
  const task: RecentTask = {
    id: command.id,
    lastEvent: {
      date: command.registeredTime,
      name: 'requested',
    },
    misc: command.misc,
    name,
    serviceAddress: command.serviceAddress,
    startTime: command.registeredTime,
    state: command.state as CommandState,
    type: command.kind,
    username: command.user.username,
  };
  return task;
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
    startTime: experiment.startTime,
    state: experiment.state,
    url: experiment.url,
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
  [RunState.Errored]: 'Errored',
  [RunState.Paused]: 'Paused',
  [RunState.StoppingCanceled]: 'Canceling',
  [RunState.StoppingCompleted]: 'Completing',
  [RunState.StoppingError]: 'Erroring',
  [RunState.Unspecified]: 'Unspecified',
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

export const checkpointStateToLabel: {[key in CheckpointState]: string} = {
  [CheckpointState.Active]: 'Active',
  [CheckpointState.Completed]: 'Completed',
  [CheckpointState.Error]: 'Error',
  [CheckpointState.Deleted]: 'Deleted',
  [CheckpointState.Unspecified]: 'Unspecified',
};

export const isTaskKillable = (task: AnyTask | ExperimentItem): boolean => {
  return killableRunStates.includes(task.state as RunState)
    || killableCmdStates.includes(task.state as CommandState);
};

export function stateToLabel(state: RunState | CommandState | CheckpointState): string {
  return runStateToLabel[state as RunState]
  || commandStateToLabel[state as CommandState]
  || checkpointStateToLabel[state as CheckpointState];
}

export const commandTypeToLabel: {[key in CommandType]: string} = {
  [CommandType.Command]: 'Command',
  [CommandType.Notebook]: 'Notebook',
  [CommandType.Shell]: 'Shell',
  [CommandType.Tensorboard]: 'Tensorboard',
};

/*
 * `keyof any` is short for "string | number | symbol"
 * since an object key can be any of those types, our key can too
 * in TS 3.0+, putting just "string" raises an error
 */
/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export function hasKey<O>(obj: O, key: keyof any): key is keyof O {
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

interface TrialDurations {
  train: number;
  checkpoint: number;
  validation: number;
}

export const trialDurationsStep = (steps: Step[]): TrialDurations => {
  const initialDurations: TrialDurations = {
    checkpoint: 0,
    train: 0,
    validation: 0,
  };

  return steps.reduce((acc: TrialDurations, cur: Step) => {
    acc.train += getDuration(cur);
    if (cur.checkpoint) acc.checkpoint += getDuration(cur.checkpoint);
    if (cur.validation) acc.validation += getDuration(cur.validation);
    return acc;
  }, initialDurations);
};

export const trialDurations = (wlWrappers: WorkloadWrapper[]): TrialDurations => {
  const initialDurations: TrialDurations = {
    checkpoint: 0,
    train: 0,
    validation: 0,
  };

  return wlWrappers.reduce((acc: TrialDurations, cur: WorkloadWrapper) => {
    if (cur.training) acc.train += getDuration(cur.training);
    if (cur.checkpoint) acc.checkpoint += getDuration(cur.checkpoint);
    if (cur.validation) acc.validation += getDuration(cur.validation);
    return acc;
  }, initialDurations);
};

/* Experiment Config */
export const trialHParamsToExperimentHParams = (hParams: Record<string, unknown>)
: ExperimentHyperParams => {
  const experimentHParams: ExperimentHyperParams = {};
  Object.entries(hParams).forEach(([ param, value ]) => {
    experimentHParams[param] = {
      type: 'const',
      val: value,
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
export const tsbMatchesSource = (tensorboard: Command, source: TBSource): boolean => {
  source.ids.sort();
  switch (source.type) {
    case TBSourceType.Experiment:
      tensorboard.misc?.experimentIds?.sort();
      return isEqual(tensorboard.misc?.experimentIds, source.ids);
    case TBSourceType.Trial:
      tensorboard.misc?.trialIds?.sort();
      return isEqual(tensorboard.misc?.trialIds, source.ids);
    default:
      return false;
  }
};
