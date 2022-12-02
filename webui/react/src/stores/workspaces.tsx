import React, { createContext, PropsWithChildren, useCallback, useContext, useState } from 'react';

import { getWorkspaces } from 'services/api';
import { Workspace } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

type WorkspacesContext = {
  updateWorkspaces: (fn: (ws: Loadable<Workspace[]>) => Loadable<Workspace[]>) => void;
  workspaces: Loadable<Workspace[]>;
};

const WorkspacesContext = createContext<WorkspacesContext | null>(null);

export const WorkspacesProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const [state, setState] = useState<Loadable<Workspace[]>>(NotLoaded);
  return (
    <WorkspacesContext.Provider value={{ updateWorkspaces: setState, workspaces: state }}>
      {children}
    </WorkspacesContext.Provider>
  );
};

export const useFetchWorkspaces = (canceler: AbortController): (() => Promise<void>) => {
  const context = useContext(WorkspacesContext);
  if (context === null) {
    throw new Error('Attempted to use useFetchWorkspaces outside of Workspace Context');
  }
  const { updateWorkspaces } = context;

  return useCallback(async (): Promise<void> => {
    try {
      const response = await getWorkspaces({}, { signal: canceler.signal });
      updateWorkspaces(() => Loaded(response.workspaces));
    } catch (e) {
      handleError(e);
    }
  }, [canceler, updateWorkspaces]);
};

export const useEnsureWorkspacesFetched = (canceler: AbortController): (() => Promise<void>) => {
  const context = useContext(WorkspacesContext);
  if (context === null) {
    throw new Error('Attempted to use useEnsureFetchWorkspaces outside of Workspace Context');
  }
  const { workspaces, updateWorkspaces } = context;

  return useCallback(async (): Promise<void> => {
    if (workspaces !== NotLoaded) return;
    try {
      const response = await getWorkspaces({}, { signal: canceler.signal });
      updateWorkspaces(() => Loaded(response.workspaces));
    } catch (e) {
      handleError(e);
    }
  }, [canceler, workspaces, updateWorkspaces]);
};

export const useWorkspaces = (): Loadable<Workspace[]> => {
  const context = useContext(WorkspacesContext);
  if (context === null) {
    throw new Error('Attempted to use useWorkspaces outside of Workspace Context');
  }
  const { workspaces } = context;

  return workspaces;
};
