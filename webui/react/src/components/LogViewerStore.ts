import { TrialLog } from '../types';

export interface ViewerLog extends TrialLog {
  formattedTime: string;
}

export enum LogStoreActionType {
  Clear,
  Append,
  Prepend,
}

export type LogStoreAction =
  | { type: LogStoreActionType.Clear; }
  | { type: LogStoreActionType.Append; value: ViewerLog[] }
  | { type: LogStoreActionType.Prepend; value: ViewerLog[] };

const logUniqueFilter = (log: ViewerLog, index: number, self: ViewerLog[]) => {
  return self.map(mapObj => mapObj.id).indexOf(log.id) === index;
};

export const logStoreReducer = (state: ViewerLog[], action: LogStoreAction): ViewerLog[] => {
  switch (action.type) {
    case LogStoreActionType.Clear: {
      return [];
    }
    case LogStoreActionType.Append:
      return [
        ...state,
        ...action.value,
      ].filter(logUniqueFilter);
    case LogStoreActionType.Prepend:
      return [
        ...action.value,
        ...state,
      ].filter(logUniqueFilter);
    default:
      return state;
  }
};
