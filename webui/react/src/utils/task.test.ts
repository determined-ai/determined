import { CommandState, CommandTask, CommandType, ExperimentTask, RunState, Task } from 'types';

import { canBeOpened, isExperimentTask } from './task';

const SampleTask: Task = { id: '', name: '', resourcePool: '', startTime: '' };
const SampleExperimentTask: ExperimentTask = {
  ...SampleTask,
  archived: false,
  parentArchived: false,
  projectId: 0,
  resourcePool: '',
  state: 'ACTIVE' as RunState,
  userId: 345,
  username: '',
  workspaceId: 0,
};
const SampleCommandTask: CommandTask = {
  ...SampleTask,
  resourcePool: '',
  state: 'PENDING' as CommandState,
  type: 'COMMAND' as CommandType,
  userId: 345,
};

describe('isExperimentTask', () => {
  it('Experiment Task', () => {
    expect(isExperimentTask(SampleExperimentTask)).toBe(true);
  });
  it('Command Task', () => {
    expect(isExperimentTask(SampleCommandTask)).toBe(false);
  });
});

describe('canBeOpened', () => {
  it('Experiment Task', () => {
    expect(canBeOpened(SampleExperimentTask)).toBe(true);
  });
  it('Terminated Command Task', () => {
    expect(canBeOpened({ ...SampleCommandTask, state: 'TERMINATED' as CommandState })).toBe(false);
  });
  it('Command Task without service address', () => {
    expect(canBeOpened(SampleCommandTask)).toBe(false);
  });
  it('Command Task with service address', () => {
    expect(canBeOpened({ ...SampleCommandTask, serviceAddress: 'test' })).toBe(true);
  });
});
