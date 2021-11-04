import React, { useReducer } from 'react';

import { DATETIME_FORMAT } from 'components/LogViewerEntry';
import { Log, RecordKey } from 'types';
import { clone } from 'utils/data';
import { formatDatetime } from 'utils/date';

export enum ActionType {
  Append = 'append',
  Clear = 'clear',
  Prepend = 'prepend',
}

interface StoreLog extends Log {
  formattedTime: string;
}

interface LogStore {
  logs: StoreLog[];
  map: Hash;
}

type Hash = Record<RecordKey, boolean>;

type Action =
  | { type: ActionType.Clear; }
  | { type: ActionType.Append; value: Log[] }
  | { type: ActionType.Prepend; value: Log[] };

export const initLogStore = {
  logs: [],
  map: {},
};

const filterLogs = (logs: Log[], map: Hash): { logs: StoreLog[], map: Hash } => {
  const newLogs = logs
    .filter(log => {
      const isDuplicate = map[log.id];
      const isTqdm = log.message.includes('\r');
      if (!isDuplicate && !isTqdm) {
        map[log.id] = true;
        return true;
      }
      return false;
    })
    .map(log => {
      const formattedTime = log.time ? formatDatetime(log.time, DATETIME_FORMAT) : '';
      return { ...log, formattedTime };
    })
    .sort((logA, logB) => {
      const logATime = logA.time || '';
      const logBTime = logB.time || '';
      return logATime.localeCompare(logBTime);
    });
  return { logs: newLogs, map };
};

const reducer = (state: LogStore, action: Action): LogStore => {
  switch (action.type) {
    case ActionType.Append: {
      const { logs, map } = filterLogs(action.value, state.map);
      return {
        ...state,
        logs: [ ...state.logs, ...logs ],
        map,
      };
    }
    case ActionType.Clear:
      return clone(initLogStore);
    case ActionType.Prepend: {
      const { logs, map } = filterLogs(action.value, state.map);
      return {
        ...state,
        logs: [ ...logs, ...state.logs ],
        map,
      };
    }
    default:
      return state;
  }
};

const useLogStore = (): [ Log[], React.Dispatch<Action> ] => {
  const [ store, dispatch ] = useReducer(reducer, clone(initLogStore));
  return [ store.logs, dispatch ];
};

export default useLogStore;
