import { CommandType, JobType } from 'types';

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

  describe('moveJobToPositionUpdate', () => {
    const jobId = 'jobId';
    it('should return the correct update', () => {
      const position = 1;
      expect(utils.moveJobToPositionUpdate(jobId, position)).toEqual({
        jobId,
        queuePosition: position - 1,
      });
    });
    it('should throw given invalid position input', () => {
      expect(() => utils.moveJobToPositionUpdate(jobId, -1))
        .toThrow('Invalid queue position: -1');
      expect(() => utils.moveJobToPositionUpdate(jobId, 0.3))
        .toThrow('Invalid queue position: 0.3');

    });
  });
});
