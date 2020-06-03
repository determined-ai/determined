import { CommandState, RecentTask, RunState, Task, TaskType } from 'types';

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export function getRandomElementOfEnum(e: any): any {
  const keys = Object.keys(e);
  const index: number = Math.floor(Math.random() * keys.length);
  return e[keys[index]];
}

export function generateTasks(): RecentTask[] {
  const runStates = new Array(10).fill(0).map(() => getRandomElementOfEnum(RunState));
  const cmdStates = new Array(10).fill(0).map(() => getRandomElementOfEnum(CommandState));
  const states = [ ...runStates, ...cmdStates ];
  const startTime = (new Date()).toString();
  return states.map((state, idx) => {
    const progress = Math.random();
    const props = {
      id: `${idx}`,
      lastEvent: {
        date: startTime,
        name: 'opened',
      },
      ownerId: Math.floor(Math.random() * 100),
      progress,
      startTime,
      state: state as RunState | CommandState,
      title: `${idx}`,
      type: getRandomElementOfEnum(TaskType) as TaskType,
      url: '#',
    };
    return props;
  });
}

// FIXME should check for type specific conditions eg tensorboard is terminated
export const canBeOpened = (task: Task): boolean => !!task.url;
