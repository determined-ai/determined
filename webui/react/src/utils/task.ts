import { killableCommandStates, killableRunStates, terminalCommandStates } from 'constants/states';
import { paths } from 'routes/utils';
import { LaunchTensorBoardParams } from 'services/types';
import * as Type from 'types';

import { isEqual } from './data';

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
  { id: 0, username: 'admin' },
  { id: 1, username: 'determined' },
  { id: 2, username: 'hamid' },
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
    progress,
    state: state as Type.RunState,
    url: '#',
    username: user.username,
  };
}

export function generateCommandTask(idx: number): Type.RecentCommandTask {
  const state = getRandomElementOfEnum(Type.CommandState);
  const task = generateTask(idx);
  const user = sampleUsers.random();
  return {
    ...task,
    state: state as Type.CommandState,
    type: getRandomElementOfEnum(Type.CommandType),
    username: user.username,
  };
}

export const generateOldExperiment = (id = 1): Type.ExperimentOld => {
  const experimentTask = generateExperimentTask(id);
  const user = sampleUsers[Math.floor(Math.random() * sampleUsers.length)];
  const config = {
    name: experimentTask.name,
    resources: {},
    searcher: { metric: 'val_error', name: 'single', smallerIsBetter: true },
  };
  const exp = generateExperiments(1)[0];
  return {
    ...exp,
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
      name: experimentTask.name,
      resources: {},
      searcher: { metric: 'val_error', name: 'single', smallerIsBetter: true },
    },
    configRaw: config,
    hyperparameters: {},
    id: id,
    name: experimentTask.name,
    username: user.username,
  } as Type.ExperimentOld;
};

export const generateOldExperiments = (count = 10): Type.ExperimentOld[] => {
  return new Array(Math.floor(count))
    .fill(null)
    .map((_, idx) => generateOldExperiment(idx));
};

export const generateExperiments = (count = 30): Type.ExperimentItem[] => {
  return new Array(Math.floor(count))
    .fill(null)
    .map((_, idx) => {
      const experimentTask = generateExperimentTask(idx);
      const user = sampleUsers.random();
      return {
        ...experimentTask,
        id: idx,
        jobId: idx.toString(),
        labels: [],
        name: experimentTask.name,
        numTrials: Math.round(Math.random() * 60000),
        resourcePool: `ResourcePool-${Math.floor(Math.random() * 3)}`,
        searcherType: 'single',
        username: user.username,
      } as Type.ExperimentItem;
    });
};

export const generateTasks = (count = 10): Type.RecentTask[] => {
  return new Array(Math.floor(count)).fill(0)
    .map((_, idx) => {
      if (Math.random() > 0.5) {
        return generateCommandTask(idx);
      } else {
        return generateExperimentTask(idx);
      }
    });
};

// Differentiate Task from Experiment.
export const isCommandTask = (obj: Type.Command | Type.CommandTask): obj is Type.CommandTask => {
  return 'type' in obj;
};

export const isExperimentTask = (task: Type.AnyTask): task is Type.ExperimentTask => {
  return ('archived' in task) && !('type' in task);
};

export const isTaskKillable = (task: Type.AnyTask | Type.ExperimentItem): boolean => {
  return killableRunStates.includes(task.state as Type.RunState)
    || killableCommandStates.includes(task.state as Type.CommandState);
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
  task: T, users?: string[],
): boolean => {
  if (!Array.isArray(users) || users.length === 0 || users[0] === Type.ALL_VALUE) return true;
  return users.findIndex(user => task.username === user) !== -1;
};

export const filterTasks = <
  T extends Type.CommandType | Type.TaskType = Type.TaskType,
  A extends Type.CommandTask | Type.AnyTask = Type.AnyTask
>(
  tasks: A[], filters: Type.TaskFilters<T>, users: Type.User[], search = '',
): A[] => {
  return tasks
    .filter(task => {
      const isExperiment = isExperimentTask(task);
      const type = isExperiment ? Type.TaskType.Experiment : (task as Type.CommandTask).type;
      return (!Array.isArray(filters.types) || filters.types.includes(type as T)) &&
        matchesUser<A>(task, filters.users) &&
        matchesState<A>(task, filters.states || []) &&
        matchesSearch<A>(task, search) &&
        (!isExperiment || !(task as Type.ExperimentTask).archived);
    })
    .filter(task => matchesSearch<A>(task, search));
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

export const taskFromExperiment = (experiment: Type.ExperimentItem): Type.RecentExperimentTask => {
  const lastEvent = experiment.endTime ?
    { date: experiment.endTime, name: 'finished' } :
    { date: experiment.startTime, name: 'requested' };
  const task: Type.RecentTask = {
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

// Checks whether tensorboard source matches a given source list.
export const tensorBoardMatchesSource = (
  tensorBoard: Type.CommandTask,
  source: LaunchTensorBoardParams,
): boolean => {
  if (source.experimentIds) {
    source.experimentIds?.sort();
    tensorBoard.misc?.experimentIds?.sort();

    if (isEqual(tensorBoard.misc?.experimentIds, source.experimentIds)) {
      return true;
    }
  }

  if (source.trialIds) {
    source.trialIds?.sort();
    tensorBoard.misc?.trialIds?.sort();

    if (isEqual(tensorBoard.misc?.trialIds, source.trialIds)) {
      return true;
    }
  }

  return false;
};
