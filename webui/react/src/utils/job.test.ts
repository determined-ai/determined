import * as Api from 'services/api-ts-sdk';
import { CommandType, Job, JobType } from 'types';
import * as utils from 'utils/job';

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
      { jobId: 'jobId3', summary: { jobsAhead: 2 } },
      { jobId: 'jobId4', summary: { jobsAhead: 3 } },
    ] as Job[];
    it('should avoid updating if the position is the same', () => {
      const position = 1;
      expect(utils.moveJobToPositionUpdate(jobs, jobId, position)).toBeUndefined();
    });

    it('should use behindOf for putting the job last', () => {
      const expected: Api.V1QueueControl = {
        behindOf: jobs.last().jobId,
        jobId,
      };
      expect(utils.moveJobToPositionUpdate(jobs, jobId, jobs.length)).toEqual(expected);
    });

    it('should throw given invalid position input', () => {
      expect(() => utils.moveJobToPositionUpdate(jobs, jobId, -1)).toThrow('Moving job failed');
      expect(() => utils.moveJobToPositionUpdate(jobs, jobId, 0.3)).toThrow('Moving job failed');
    });

    it('should work on middle of the job queue for moving up', () => {
      const id = 'jobId3';
      const expected: Api.V1QueueControl = {
        aheadOf: 'jobId2',
        jobId: id,
      };
      expect(utils.moveJobToPositionUpdate(jobs, id, 2)).toEqual(expected);
    });

    it('should work on middle of the job queue for moving down', () => {
      const id = 'jobId2';
      const expected: Api.V1QueueControl = {
        behindOf: 'jobId3',
        jobId: id,
      };
      expect(utils.moveJobToPositionUpdate(jobs, id, 3)).toEqual(expected);
    });
  });
});
