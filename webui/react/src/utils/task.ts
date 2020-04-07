import { CommandState, RecentTask, RunState, TaskType } from 'types';

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

  return states.map((state, idx) => {
    const progress = Math.random();
    const props = {
      id: `${idx}`,
      lastEvent: {
        date: (new Date()).toString(),
        name: 'opened',
      },
      ownerId: Math.floor(Math.random() * 100),
      progress,
      state: state as RunState | CommandState,
      title: `${idx}`,
      type: getRandomElementOfEnum(TaskType) as TaskType,
      url: '#',
    };
    return props;
  });
}
