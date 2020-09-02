import {
  ALL_VALUE, AnyTask, CommandState, CommandTask, CommandType,
  ExperimentItem, ExperimentOld, ExperimentTask, RecentCommandTask, RecentEvent,
  RecentExperimentTask, RecentTask, RunState, Task, TaskFilters, TaskType, User,
} from 'types';
import { terminalCommandStates } from 'utils/types';

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export function getRandomElementOfEnum(e: any): any {
  const keys = Object.keys(e);
  const index: number = Math.floor(Math.random() * keys.length);
  return e[keys[index]];
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
    startTime,
    url: '#',
  };
}

export function generateExperimentTask(idx: number): RecentExperimentTask {
  const state = getRandomElementOfEnum(RunState);
  const task = generateTask(idx);
  const progress = Math.random();
  const user = sampleUsers[Math.floor(Math.random() * sampleUsers.length)];
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
  const user = sampleUsers[Math.floor(Math.random() * sampleUsers.length)];
  return {
    ...task,
    state: state as CommandState,
    type: getRandomElementOfEnum(CommandType),
    username: user.username,
  };
}

export const generateOldExperiments = (count = 10): ExperimentOld[] => {
  return new Array(Math.floor(count))
    .fill(null)
    .map((_, idx) => {
      const experimentTask = generateExperimentTask(idx);
      const user = sampleUsers[Math.floor(Math.random() * sampleUsers.length)];
      const config = {
        description: experimentTask.name,
        resources: {},
        searcher: { metric: 'val_error', smallerIsBetter: true },
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
          description: experimentTask.name,
          resources: {},
          searcher: { metric: 'val_error', smallerIsBetter: true },
        },
        configRaw: config,
        id: idx,
        name: experimentTask.name,
        userId: user.id,
        username: user.username,
      } as ExperimentOld;
    });
};

export const generateExperiments = (count = 30): ExperimentItem[] => {
  return new Array(Math.floor(count))
    .fill(null)
    .map((_, idx) => {
      const experimentTask = generateExperimentTask(idx);
      const user = sampleUsers[Math.floor(Math.random() * sampleUsers.length)];
      return {
        ...experimentTask,
        id: idx,
        labels: [],
        name: experimentTask.name,
        numTrials: Math.round(Math.random() * 60000),
        userId: user.id,
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

export const canBeOpened = (task: AnyTask): boolean => {
  if (isExperimentTask(task)) return true;
  if (terminalCommandStates.has(task.state)) return false;
  return !!task.url;
};

const matchesSearch = <T extends AnyTask | ExperimentItem>(task: T, search = ''): boolean => {
  if (!search) return true;
  return task.id.toString().indexOf(search) !== -1 || task.name.indexOf(search) !== -1;
};

const matchesState = <T extends AnyTask | ExperimentItem>(task: T, states: string[]): boolean => {
  if (states[0] === ALL_VALUE) return true;

  const targetStateRun = states[0] as RunState;
  const targetStateCmd = states[0] as CommandState;

  return [ targetStateRun, targetStateCmd ].includes(task.state);
};

const matchesUser = <T extends AnyTask | ExperimentItem>(
  task: T, users: User[], username?: string,
): boolean => {
  if (!username) return true;
  const selectedUser = users.find(u => u.username === username);
  return !!selectedUser && (task.username === username);
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
    .filter(task => matchesSearch<A>(task, search));
};
