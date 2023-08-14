import _ from 'lodash';

import { killableCommandStates, killableRunStates, terminalCommandStates } from 'constants/states';
import { LaunchTensorBoardParams } from 'services/types';
import * as Type from 'types';
import { CommandState, RunState, State } from 'types';

import { runStateSortValues } from './experiment';

export const canBeOpened = (task: Type.AnyTask): boolean => {
  if (isExperimentTask(task)) return true;
  if (terminalCommandStates.has(task.state)) return false;
  return !!task.serviceAddress;
};

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export function getRandomElementOfEnum(e: any): any {
  const keys = Object.keys(e);
  return e[keys.random()];
}

export const sampleUsers = [
  { displayName: '', id: 0, username: 'admin' },
  { displayName: '', id: 1, username: 'determined' },
  { displayName: '', id: 2, username: 'hamid' },
];

function generateTask(idx: number): Type.Task & Type.RecentEvent {
  const now = Date.now();
  const range = Math.random() * 2 * 356 * 24 * 60 * 60 * 1000;
  const startTime = new Date(now - range).toString();
  return {
    id: `${idx}`,
    lastEvent: {
      date: startTime,
      name: 'opened',
    },
    name: `${idx}`,
    resourcePool: `ResourcePool-${Math.floor(Math.random() * 3)}`,
    startTime,
    url: '#',
  };
}

export function generateExperimentTask(idx: number): Type.RecentExperimentTask {
  const state = getRandomElementOfEnum(Type.RunState);
  const task = generateTask(idx);
  const progress = Math.random();
  const user = sampleUsers.random();
  return {
    ...task,
    archived: false,
    parentArchived: false,
    progress,
    projectId: 1,
    state: state as Type.RunState,
    url: '#',
    userId: user.id,
    username: user.username,
    workspaceId: 1,
  };
}

export const generateExperiment = (id = 1): Type.ExperimentItem => {
  const experimentTask = generateExperimentTask(id);
  const user = sampleUsers.random();
  const config = {
    name: experimentTask.name,
    resources: {},
    searcher: { metric: 'val_error', name: 'single', smallerIsBetter: true },
  };
  return {
    ...experimentTask,
    config: {
      checkpointPolicy: 'best',
      checkpointStorage: {
        hostPath: '/tmp',
        saveExperimentBest: 0,
        saveTrialBest: 1,
        saveTrialLatest: 1,
        storagePath: 'determined-integration-checkpoints',
        type: 'shared_fs',
      },
      dataLayer: { type: 'shared_fs' },
      hyperparameters: {},
      maxRestarts: 5,
      name: experimentTask.name,
      resources: {},
      searcher: { metric: 'val_error', name: 'single', smallerIsBetter: true },
    },
    configRaw: config,
    hyperparameters: {},
    id: id,
    jobId: id.toString(),
    labels: [],
    name: experimentTask.name,
    numTrials: Math.round(Math.random() * 60000),
    projectId: 1,
    resourcePool: `ResourcePool-${Math.floor(Math.random() * 3)}`,
    searcherType: 'single',
    userId: user.id,
    username: user.username,
  } as Type.ExperimentItem;
};

export const generateExperiments = (count = 30): Type.ExperimentItem[] => {
  return new Array(Math.floor(count)).fill(null).map((_, idx) => generateExperiment(idx));
};

// Differentiate Task from Experiment.
export const isCommandTask = (obj: Type.Command | Type.CommandTask): obj is Type.CommandTask => {
  return 'type' in obj;
};

export const isExperimentTask = (task: Type.AnyTask): task is Type.ExperimentTask => {
  return 'archived' in task && !('type' in task);
};

export const isTaskKillable = (
  task: Type.AnyTask | Type.ExperimentItem,
  canModifyWorkspaceNSC: boolean,
): boolean => {
  return (
    canModifyWorkspaceNSC &&
    (killableRunStates.includes(task.state as Type.RunState) ||
      killableCommandStates.includes(task.state as Type.CommandState))
  );
};

const matchesSearch = <T extends Type.AnyTask | Type.ExperimentItem>(
  task: T,
  search = '',
): boolean => {
  if (!search) return true;
  return task.id.toString().indexOf(search) !== -1 || task.name.indexOf(search) !== -1;
};

const matchesState = <T extends Type.AnyTask | Type.ExperimentItem>(
  task: T,
  states: string[],
): boolean => {
  if (!Array.isArray(states) || states.length === 0 || states[0] === Type.ALL_VALUE) return true;
  return states.includes(task.state as string);
};

const matchesUser = <T extends Type.AnyTask | Type.ExperimentItem>(
  task: T,
  users?: string[],
): boolean => {
  if (!Array.isArray(users) || users.length === 0 || users[0] === Type.ALL_VALUE) return true;
  return users.findIndex((user) => task.userId === parseInt(user)) !== -1;
};

const matchesWorkspace = <T extends Type.AnyTask | Type.ExperimentItem>(
  task: T,
  workspaces?: string[],
): boolean => {
  if (!Array.isArray(workspaces) || workspaces.length === 0 || workspaces[0] === Type.ALL_VALUE)
    return true;
  return workspaces.findIndex((workspace) => task.workspaceId === parseInt(workspace)) !== -1;
};

export const filterTasks = <
  T extends Type.CommandType | Type.TaskType = Type.TaskType,
  A extends Type.CommandTask | Type.AnyTask = Type.AnyTask,
>(
  tasks: A[],
  filters: Type.TaskFilters<T>,
  users: Type.User[],
  search = '',
): A[] => {
  return tasks
    .filter((task) => {
      const isExperiment = isExperimentTask(task);
      const type = isExperiment ? Type.TaskType.Experiment : (task as Type.CommandTask).type;
      return (
        (!Array.isArray(filters.types) || filters.types.includes(type as T)) &&
        matchesUser<A>(task, filters.users) &&
        matchesWorkspace<A>(task, filters.workspaces) &&
        matchesState<A>(task, filters.states || []) &&
        matchesSearch<A>(task, search) &&
        (!isExperiment || !(task as Type.ExperimentTask).archived)
      );
    })
    .filter((task) => matchesSearch<A>(task, search));
};

/* Conversions to Tasks */

export const taskFromCommandTask = (command: Type.CommandTask): Type.RecentCommandTask => {
  return {
    ...command,
    lastEvent: {
      date: command.startTime,
      name: 'requested',
    },
  };
};

// Checks whether tensorboard source matches a given source list.
export const tensorBoardMatchesSource = (
  tensorBoard: Type.CommandTask,
  source: LaunchTensorBoardParams,
): boolean => {
  if (source.experimentIds) {
    source.experimentIds?.sort();
    tensorBoard.misc?.experimentIds?.sort();

    if (_.isEqual(tensorBoard.misc?.experimentIds, source.experimentIds)) {
      return true;
    }
  }

  if (source.trialIds) {
    source.trialIds?.sort();
    tensorBoard.misc?.trialIds?.sort();

    if (_.isEqual(tensorBoard.misc?.trialIds, source.trialIds)) {
      return true;
    }
  }

  return false;
};

const commandStateSortOrder: CommandState[] = [
  CommandState.Pulling,
  CommandState.Starting,
  CommandState.Running,
  CommandState.Waiting,
  CommandState.Terminating,
  CommandState.Terminated,
];

const commandStateSortValues: Map<CommandState, number> = new Map(
  commandStateSortOrder.map((state, idx) => [state, idx]),
);

export const commandStateSorter = (a: CommandState, b: CommandState): number => {
  return (commandStateSortValues.get(a) || 0) - (commandStateSortValues.get(b) || 0);
};

export const taskStateSorter = (a: State, b: State): number => {
  // FIXME this is O(n) we can do it in constant time.
  // What is the right typescript way of doing it?
  const aValue = Object.values(RunState).includes(a as RunState)
    ? runStateSortValues.get(a as RunState) || 0
    : commandStateSortValues.get(a as CommandState) || 0;
  const bValue = Object.values(RunState).includes(b as RunState)
    ? runStateSortValues.get(b as RunState) || 0
    : commandStateSortValues.get(b as CommandState) || 0;
  return aValue - bValue;
};
