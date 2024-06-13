import { FlatRun, TrialDetails } from 'types';

export const isRun = (trial?: TrialDetails | FlatRun): trial is FlatRun => {
  if (!trial) return false;
  return 'checkpointSize' in trial;
};
