import React, { createContext, PropsWithChildren, useCallback, useContext, useState } from 'react';

import { getActiveTasks } from 'services/api';
import { TaskCounts } from 'types';
import handleError from 'utils/error';

type TasksContext = {
  activeTasks: TaskCounts;
  updateActiveTasks: (fn: (ws: TaskCounts) => TaskCounts) => void;
};

const TasksContext = createContext<TasksContext | null>(null);

export const TasksProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const [state, setState] = useState<TaskCounts>({
    commands: 0,
    notebooks: 0,
    shells: 0,
    tensorboards: 0,
  });
  return (
    <TasksContext.Provider value={{ activeTasks: state, updateActiveTasks: setState }}>
      {children}
    </TasksContext.Provider>
  );
};

export const useFetchActiveTasks = (canceler: AbortController): (() => Promise<void>) => {
  const context = useContext(TasksContext);
  if (context === null) {
    throw new Error('Attempted to use useFetchTasks outside of Task Context');
  }
  const { updateActiveTasks } = context;

  return useCallback(async (): Promise<void> => {
    try {
      const response = await getActiveTasks({}, { signal: canceler.signal });
      updateActiveTasks(() => response);
    } catch (e) {
      handleError(e);
    }
  }, [canceler, updateActiveTasks]);
};

export const useActiveTasks = (): TaskCounts => {
  const context = useContext(TasksContext);
  if (context === null) {
    throw new Error('Attempted to use useActiveTasks outside of Task Context');
  }
  const { activeTasks } = context;

  return activeTasks;
};
