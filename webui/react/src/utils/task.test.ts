import { CommandState, CommandTask, CommandType, ExperimentTask, RunState, Task } from 'types';

import { canBeOpened, isExperimentTask } from './task';

const SampleTask: Task = { id: '', name: '', resourcePool: '', startTime: '' };
const SampleExperimentTask: ExperimentTask = {
  ...SampleTask,
  archived: false,
  resourcePool: '',
  state: 'ACTIVE' as RunState,
  username: '',
};
const SampleCommandTask: CommandTask = {
  ...SampleTask,
  resourcePool: '',
  state: 'PENDING' as CommandState,
  type: 'COMMAND' as CommandType,
  username: '',
};

describe('isExperimentTask', () => {
  it('Experiment Task', () => {
    expect(isExperimentTask(SampleExperimentTask)).toStrictEqual(true);
  });
  it('Command Task', () => {
    expect(isExperimentTask(SampleCommandTask)).toStrictEqual(false);
  });
});

describe('canBeOpened', () => {
  it('Experiment Task', () => {
    expect(canBeOpened(SampleExperimentTask)).toStrictEqual(true);
  });
  it('Terminated Command Task', () => {
    expect(canBeOpened({ ...SampleCommandTask, state: 'TERMINATED' as CommandState }))
      .toStrictEqual(false);
  });
  it('Command Task without service address', () => {
    expect(canBeOpened(SampleCommandTask)).toStrictEqual(false);
  });
  it('Command Task with service address', () => {
    expect(canBeOpened({ ...SampleCommandTask, serviceAddress: '' })).toStrictEqual(true);
  });
});
