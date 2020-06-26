import {
  ALL_VALUE, AnyTask, CommandState, CommandTask, CommandType, ExperimentTask, RecentCommandTask,
  RecentEvent, RecentExperimentTask, RecentTask, RunState, Task, TaskFilters, TaskType,
  terminalCommandStates, User,
} from 'types';

import { isExperiment } from './types';

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export function getRandomElementOfEnum(e: any): any {
  const keys = Object.keys(e);
  const index: number = Math.floor(Math.random() * keys.length);
  return e[keys[index]];
}

const sampleUsers = [
  {
    id: 0,
    username: 'admin',
  },
  {
    id: 1,
    username: 'determined',
  },
  {
    id: 2,
    username: 'hamid',
  },
];

function generateTask(idx: number): Task & RecentEvent {
  const startTime = (Date.now()).toString();
  const user = sampleUsers[Math.floor(Math.random() * sampleUsers.length)];
  return {
    id: `${idx}`,
    lastEvent: {
      date: startTime,
      name: 'opened',
    },
    ownerId: user.id,
    startTime,
    title: `${idx}`,
    url: '#',
  };
}

export function generateExperimentTask(idx: number): RecentExperimentTask {
  const state = getRandomElementOfEnum(RunState);
  const task = generateTask(idx);
  const progress = Math.random();
  return {
    archived: false,
    ... task,
    progress,
    state: state as RunState,
  };
}

export function generateCommandTask(idx: number): RecentCommandTask {
  const state = getRandomElementOfEnum(CommandState);
  const task = generateTask(idx);
  let username = sampleUsers.find(user => user.id === task.ownerId)?.username;
  if (!username)
    username = sampleUsers[Math.floor(Math.random() * sampleUsers.length)].username;
  return {
    ...task,
    state: state as CommandState,
    type: getRandomElementOfEnum(CommandType),
    username,
  };
}

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

export const canBeOpened = (task: AnyTask): boolean => {
  if (!isExperimentTask(task) && task.state in terminalCommandStates) return false;
  if (isExperiment(task)) return true;
  return !!task.url;
};

const matchesSearch = <T extends AnyTask>(task: T, search = ''): boolean => {
  if (!search) return true;
  return task.id.indexOf(search) !== -1 || task.title.indexOf(search) !== -1;
};

const matchesState = <T extends AnyTask>(task: T, states: string[]): boolean => {
  if (states[0] === ALL_VALUE) return true;

  const targetStateRun = states[0] as RunState;
  const targetStateCmd = states[0] as CommandState;

  return [ targetStateRun, targetStateCmd ].includes(task.state);
};

const matchesUser = <T extends AnyTask>(task: T, users: User[], username?: string): boolean => {
  if (!username) return true;
  const selectedUser = users.find(u => u.username === username);
  return !!selectedUser && (task.ownerId === selectedUser.id);
};

export const filterTasks = <T extends TaskType = TaskType, A extends AnyTask = AnyTask>(
  tasks: A[], filters: TaskFilters<T>, users: User[], search = '',
): A[] => {
  const isAllTypes = !Object.values(filters.types).includes(true);
  return tasks
    .filter(task => {
      const isExperiment = isExperimentTask(task);
      const type = isExperiment ? 'Experiment' : (task as CommandTask).type;
      return (isAllTypes || filters.types[type as T]) &&
        matchesUser<A>(task, users, filters.username) &&
        matchesState<A>(task, filters.states) &&
        matchesSearch<A>(task, search) &&
        (!isExperiment || !(task as ExperimentTask).archived);
    })
    .slice(0, filters.limit);
};
