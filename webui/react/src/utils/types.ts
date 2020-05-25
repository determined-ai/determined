import {
  Command, CommandState, CommandType, Experiment,
  RecentTask, RunState, TaskType,
} from 'types';

/* Conversions to Tasks */

const commandToEventUrl = (command: Command): string | undefined => {
  if (command.kind === CommandType.Notebook) return `/notebooks/${command.id}/events`;
  if (command.kind === CommandType.Tensorboard) return `/tensorboard/${command.id}/events?tail=1`;
  return undefined;
};

const waitPageUrl = (command: Command): string | undefined => {
  const eventUrl = commandToEventUrl(command);
  const proxyUrl = command.serviceAddress;
  if (!eventUrl || !proxyUrl) return;
  const event = encodeURIComponent(eventUrl);
  const jump = encodeURIComponent(proxyUrl);
  return `/wait?event=${event}&jump=${jump}`;
};

export const commandToTask = (command: Command): RecentTask => {
  // We expect the title to be in the form of 'Type (pet-name-generated)'.
  const title = command.config.description.replace(/.*\((.*)\).*/, '$1');
  const task: RecentTask = {
    id: command.id,
    lastEvent: {
      date: command.registeredTime,
      name: 'requested',
    },
    ownerId: command.owner.id,
    state: command.state as CommandState,
    title,
    type: command.kind as unknown as TaskType,
    url: waitPageUrl(command),
  };
  return task;
};

export const experimentToTask = (experiment: Experiment): RecentTask => {
  const lastEvent = experiment.endTime ?
    { date: experiment.endTime, name: 'finished' } :
    { date: experiment.startTime, name: 'requested' };
  const task: RecentTask = {
    archived: experiment.archived,
    id: `${experiment.id}`,
    lastEvent,
    ownerId: experiment.ownerId,
    progress: typeof experiment.progress === 'number' ? experiment.progress : undefined,
    state: experiment.state as RunState,
    title: experiment.config.description,
    type: TaskType.Experiment,
    url: `/ui/experiments/${experiment.id}`,
  };
  return task;
};

export const killableRunStates = [ RunState.Active, RunState.Paused, RunState.StoppingCanceled ];
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
];

export const runStateToLabel: {[key in RunState]: string} = {
  [RunState.Active]: 'Active',
  [RunState.Canceled]: 'Canceled',
  [RunState.Completed]: 'Completed',
  [RunState.Errored]: 'Errored',
  [RunState.Paused]: 'Paused',
  [RunState.StoppingCanceled]: 'StoppingCanceled',
  [RunState.StoppingCompleted]: 'StoppingCompleted',
  [RunState.StoppingError]: 'StoppingErrored',
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

export const activeRunStates = [
  RunState.Active,
  RunState.StoppingCanceled,
  RunState.StoppingCompleted,
  RunState.StoppingError,
];

export const isTaskKillable = (task: RecentTask): boolean => {
  return killableRunStates.includes(task.state as RunState)
    || killableCmdStates.includes(task.state as CommandState);
};

export function stateToLabel(state: RunState | CommandState): string {
  return runStateToLabel[state as RunState] || commandStateToLabel[state as CommandState];
}

/*
 * `keyof any` is short for "string | number | symbol"
 * since an object key can be any of those types, our key can too
 * in TS 3.0+, putting just "string" raises an error
 */
/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export function hasKey<O>(obj: O, key: keyof any): key is keyof O {
  return key in obj;
}
