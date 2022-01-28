import * as Api from 'services/api-ts-sdk';
import { CommandType, Job, JobType } from 'types';

import * as utils from './job';

describe('Job Utilities', () => {
  describe('jobTypeIconName', () => {
    it('should support experiment and command types', () => {
      expect(utils.jobTypeIconName(JobType.EXPERIMENT)).toEqual('experiment');
      expect(utils.jobTypeIconName(JobType.NOTEBOOK)).toEqual(CommandType.JupyterLab);
    });
  });

  describe('jobTypeToCommandType', () => {
    it('should convert notebook to jupyterlab', () => {
      expect(utils.jobTypeToCommandType(JobType.NOTEBOOK)).toEqual(CommandType.JupyterLab);
    });
    it('should return undefined for non command types', () => {
      expect(utils.jobTypeToCommandType(JobType.EXPERIMENT)).toBeUndefined();
    });
  });

  // TODO more tests
  describe('moveJobToPositionUpdate', () => {
    const jobId = 'jobId1';
    const jobs = [
      { jobId: 'jobId1', summary: { jobsAhead: 0 } },
      { jobId: 'jobId2', summary: { jobsAhead: 1 } },
    ] as Job[];
    it('should avoid updating if the position is the same', () => {
      const position = 1;
      expect(utils.moveJobToPositionUpdate(jobs, jobId, position)).toBeUndefined();
    });
    it('should use behindOf for putting the job last', () => {
      const expected: Api.V1QueueControl = {
        behindOf: 'jobId2',
        jobId,
      };
      expect(utils.moveJobToPositionUpdate(jobs, jobId, 2)).toEqual(expected);
    });
    it('should throw given invalid position input', () => {
      expect(() => utils.moveJobToPositionUpdate(jobs, jobId, -1))
        .toThrow('Moving job failed');
      expect(() => utils.moveJobToPositionUpdate(jobs, jobId, 0.3))
        .toThrow('Moving job failed');

    });
  });
});
