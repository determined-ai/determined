import { detApi } from 'services/apiConfig';

export const moveJobToPosition = async (jobId: string, position: number): Promise<void> => {
  console.warn('TODO moveJobToPosition', jobId, position);
  await detApi.Internal.updateJobQueue({ updates: [] });
};
