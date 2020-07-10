import {
  AnyTask, Command, CommandState, CommandType, Experiment, RecentCommandTask,
  RecentExperimentTask, RecentTask, RunState,
} from 'types';

/* Conversions to Tasks */

const commandToEventUrl = (command: Command): string | undefined => {
  if (command.kind === CommandType.Notebook) return `/notebooks/${command.id}/events`;
  if (command.kind === CommandType.Tensorboard) return `/tensorboard/${command.id}/events?tail=1`;
  return undefined;
};

export const waitPageUrl = (command: Command): string | undefined => {
  const eventUrl = commandToEventUrl(command);
  const proxyUrl = command.serviceAddress;
  if (!eventUrl || !proxyUrl) return;
  const event = encodeURIComponent(eventUrl);
  const jump = encodeURIComponent(proxyUrl);
  return `/wait?event=${event}&jump=${jump}`;
};

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
    ownerId: command.owner.id,
    startTime: command.registeredTime,
    state: command.state as CommandState,
    type: command.kind,
    url: waitPageUrl(command),
    username: command.owner.username,
  };
  return task;
};

export const experimentToTask = (experiment: Experiment): RecentExperimentTask => {
  const lastEvent = experiment.endTime ?
    { date: experiment.endTime, name: 'finished' } :
    { date: experiment.startTime, name: 'requested' };
  const task: RecentTask = {
    archived: experiment.archived,
    id: `${experiment.id}`,
    lastEvent,
    name: experiment.config.description,
    ownerId: experiment.ownerId,
    progress: experiment.progress,
    startTime: experiment.startTime,
    state: experiment.state,
    url: `/ui/experiments/${experiment.id}`,
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

export const activeRunStates = [
  RunState.Active,
  RunState.StoppingCanceled,
  RunState.StoppingCompleted,
  RunState.StoppingError,
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

export const terminalRunStates = [
  RunState.Canceled,
  RunState.Completed,
  RunState.Errored,
  RunState.Deleted,
];

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

export const isTaskKillable = (task: AnyTask): boolean => {
  return killableRunStates.includes(task.state as RunState)
    || killableCmdStates.includes(task.state as CommandState);
};

export function stateToLabel(state: RunState | CommandState): string {
  return runStateToLabel[state as RunState] || commandStateToLabel[state as CommandState];
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
export const isExperiment = (obj: AnyTask | Experiment): obj is Experiment => {
  return 'config' in obj; // FIXME
};

// used when properties are named differently between objects.
/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const oneOfProperties = <T>(obj: any, props: string[]): T => {
  for (const prop of props) {
    if (prop in obj) return obj[prop] as T;
  }
  throw new Error('no matching property');
};
