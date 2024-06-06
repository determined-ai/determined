import { isRun } from './run';
import { generateTestExperimentData, generateTestRunData } from './tests/generateTestData';

const { trial } = generateTestExperimentData();
const run = generateTestRunData();
describe('isRun', () => {
  it('Trial', () => {
    expect(isRun(trial)).toStrictEqual(false);
  });
  it('Run', () => {
    expect(isRun(run)).toStrictEqual(true);
  });
});
