import { CommandState, RecentTask, RunState, Task, TaskType, terminalCommandStates } from 'types';

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

export function generateTasks(count = 10): RecentTask[] {
  const runStates = new Array(Math.floor(count)).fill(0)
    .map(() => getRandomElementOfEnum(RunState));
  const cmdStates = new Array(Math.ceil(count)).fill(0)
    .map(() => getRandomElementOfEnum(CommandState));
  const states = [ ...runStates, ...cmdStates ];
  const startTime = (Date.now()).toString();
  return states.map((state, idx) => {
    const progress = Math.random();
    const user = sampleUsers[Math.floor(Math.random() * sampleUsers.length)];
    const props = {
      id: `${idx}`,
      lastEvent: {
        date: startTime,
        name: 'opened',
      },
      ownerId: user.id,
      progress,
      startTime,
      state: state as RunState | CommandState,
      title: `${idx}`,
      type: getRandomElementOfEnum(TaskType) as TaskType,
      url: '#',
      username: user.username,
    };
    return props;
  });
}

export const canBeOpened = (task: Task): boolean => {
  if (task.type !== TaskType.Experiment && task.state in terminalCommandStates) {
    return false;
  }
  return !!task.url;
};
