import { paths } from 'routes/utils';
import { LaunchTensorBoardParams } from 'services/types';
import {
  ALL_VALUE, AnyTask, CommandState, CommandTask, CommandType,
  ExperimentItem, ExperimentOld, ExperimentTask, RecentCommandTask, RecentEvent,
  RecentExperimentTask, RecentTask, RunState, Task, TaskFilters, TaskType, User,
} from 'types';
import { terminalCommandStates } from 'utils/types';

import { isEqual } from './data';

export const canBeOpened = (task: AnyTask): boolean => {
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

function generateTask(idx: number): Task & RecentEvent {
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

export function generateExperimentTask(idx: number): RecentExperimentTask {
  const state = getRandomElementOfEnum(RunState);
  const task = generateTask(idx);
  const progress = Math.random();
  const user = sampleUsers.random();
  return {
    ...task,
    archived: false,
    progress,
    state: state as RunState,
    url: '#',
    username: user.username,
  };
}

export function generateCommandTask(idx: number): RecentCommandTask {
  const state = getRandomElementOfEnum(CommandState);
  const task = generateTask(idx);
  const user = sampleUsers.random();
  return {
    ...task,
    state: state as CommandState,
    type: getRandomElementOfEnum(CommandType),
    username: user.username,
  };
}

export const generateOldExperiment = (id = 1): ExperimentOld => {
  const experimentTask = generateExperimentTask(id);
  const user = sampleUsers[Math.floor(Math.random() * sampleUsers.length)];
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
      name: experimentTask.name,
      resources: {},
      searcher: { metric: 'val_error', name: 'single', smallerIsBetter: true },
    },
    configRaw: config,
    hyperparameters: {},
    id: id,
    name: experimentTask.name,
    username: user.username,
  } as ExperimentOld;
};

export const generateOldExperiments = (count = 10): ExperimentOld[] => {
  return new Array(Math.floor(count))
    .fill(null)
    .map((_, idx) => generateOldExperiment(idx));
};

export const generateExperiments = (count = 30): ExperimentItem[] => {
  return new Array(Math.floor(count))
    .fill(null)
    .map((_, idx) => {
      const experimentTask = generateExperimentTask(idx);
      const user = sampleUsers.random();
      return {
        ...experimentTask,
        id: idx,
        labels: [],
        name: experimentTask.name,
        numTrials: Math.round(Math.random() * 60000),
        resourcePool: `ResourcePool-${Math.floor(Math.random() * 3)}`,
        username: user.username,
      } as ExperimentItem;
    });
};

export const generateTasks = (count = 10): RecentTask[] => {
  return new Array(Math.floor(count)).fill(0)
    .map((_, idx) => {
      if (Math.random() > 0.5) {
        return generateCommandTask(idx);
      } else {
        return generateExperimentTask(idx);
      }
    });
};

export const isExperimentTask = (task: AnyTask): task is ExperimentTask => {
  return ('archived' in task) && !('type' in task);
};

const matchesSearch = <T extends AnyTask | ExperimentItem>(task: T, search = ''): boolean => {
  if (!search) return true;
  return task.id.toString().indexOf(search) !== -1 || task.name.indexOf(search) !== -1;
};

const matchesState = <T extends AnyTask | ExperimentItem>(task: T, states: string[]): boolean => {
  if (!Array.isArray(states) || states.length === 0 || states[0] === ALL_VALUE) return true;
  return states.includes(task.state);
};

const matchesUser = <T extends AnyTask | ExperimentItem>(
  task: T, users?: string[],
): boolean => {
  if (!Array.isArray(users) || users.length === 0 || users[0] === ALL_VALUE) return true;
  return users.findIndex(user => task.username === user) !== -1;
};

export const filterTasks = <
  T extends CommandType | TaskType = TaskType,
  A extends CommandTask | AnyTask = AnyTask
>(
  tasks: A[], filters: TaskFilters<T>, users: User[], search = '',
): A[] => {
  return tasks
    .filter(task => {
      const isExperiment = isExperimentTask(task);
      const type = isExperiment ? TaskType.Experiment : (task as CommandTask).type;
      return (!Array.isArray(filters.types) || filters.types.includes(type as T)) &&
        matchesUser<A>(task, filters.users) &&
        matchesState<A>(task, filters.states || []) &&
        matchesSearch<A>(task, search) &&
        (!isExperiment || !(task as ExperimentTask).archived);
    })
    .filter(task => matchesSearch<A>(task, search));
};

/* Conversions to Tasks */

export const taskFromCommandTask = (command: CommandTask): RecentCommandTask => {
  return {
    ...command,
    lastEvent: {
      date: command.startTime,
      name: 'requested',
    },
  };
};

export const taskFromExperiment = (experiment: ExperimentItem): RecentExperimentTask => {
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

// Checks whether tensorboard source matches a given source list.
export const tensorBoardMatchesSource = (
  tensorBoard: CommandTask,
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
