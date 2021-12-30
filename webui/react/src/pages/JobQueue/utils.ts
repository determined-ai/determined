import { V1QueueControl } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';

export const moveJobToPositionUpdate = (jobId: string, position: number): V1QueueControl => {
  console.warn('TODO moveJobToPositionUpdate', jobId, position);
  return {
    jobId,
    queuePosition: position - 1,
  };
};

export const moveJobToPosition = async (jobId: string, position: number): Promise<void> => {
  await detApi.Internal.updateJobQueue({ updates: [ moveJobToPositionUpdate(jobId, position) ] });
};
