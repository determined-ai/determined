import React, { createContext, PropsWithChildren, useCallback, useContext, useState } from 'react';

import { createWorkspace, deleteWorkspace, getWorkspaces } from 'services/api';
import { V1PostWorkspaceRequest } from 'services/api-ts-sdk';
import { GetWorkspacesParams } from 'services/types';
import { Workspace } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

type WorkspacesContext = {
  updateWorkspaces: (fn: (ws: Loadable<Workspace[]>) => Loadable<Workspace[]>) => void;
  workspaces: Loadable<Workspace[]>;
};

const WorkspacesContext = createContext<WorkspacesContext | null>(null);

export const WorkspacesProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const [workspaces, updateWorkspaces] = useState<Loadable<Workspace[]>>(NotLoaded);
  return (
    <WorkspacesContext.Provider value={{ updateWorkspaces, workspaces }}>
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
    throw new Error('Attempted to use useEnsureWorkspacesFetched outside of Workspace Context');
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

export const useWorkspaces = (params?: GetWorkspacesParams): Loadable<Workspace[]> => {
  const context = useContext(WorkspacesContext);
  if (context === null) {
    throw new Error('Attempted to use useWorkspaces outside of Workspace Context');
  }
  return Loadable.map(context.workspaces, (workspaces: Workspace[]) =>
    workspaces.filter((ws) =>
      Object.keys(params || {}).reduce<boolean>((accumulator: boolean, key) => {
        switch (key) {
          case 'archived':
            return accumulator && ws.archived === params?.archived;
          case 'pinned':
            return accumulator && ws.pinned === params?.pinned;
          case 'name':
            return accumulator && ws.name === params?.name;
          case 'users':
            return accumulator && (params?.users || []).indexOf(String(ws.userId)) > -1;
          default:
            return false;
        }
      }, true),
    ),
  );
};

export const useUpdateWorkspace = (): ((
  id: number,
  updater: (arg0: Workspace) => Workspace,
) => void) => {
  const context = useContext(WorkspacesContext);
  if (context === null) {
    throw new Error('Attempted to use useUpdateWorkspace outside of Workspace Context');
  }
  const { updateWorkspaces } = context;

  return useCallback(
    (id: number, updater: (arg0: Workspace) => Workspace): void => {
      updateWorkspaces((prev) =>
        Loadable.map(prev, (workspaces) =>
          workspaces.map((old) => (old.id === id ? updater(old) : old)),
        ),
      );
    },
    [updateWorkspaces],
  );
};

export const useCreateWorkspace = (): ((arg0: V1PostWorkspaceRequest) => Promise<Workspace>) => {
  const context = useContext(WorkspacesContext);
  if (context === null) {
    throw new Error('Attempted to use useCreateWorkspace outside of Workspace Context');
  }
  const { updateWorkspaces } = context;

  return useCallback(
    async (params: V1PostWorkspaceRequest): Promise<Workspace> => {
      const w = await createWorkspace(params);
      updateWorkspaces((prev) => Loadable.map(prev, (ws: Workspace[]) => ws.concat([w])));
      return w;
    },
    [updateWorkspaces],
  );
};

export const useDeleteWorkspace = (): ((id: number) => Promise<void>) => {
  const context = useContext(WorkspacesContext);
  if (context === null) {
    throw new Error('Attempted to use useDeleteWorkspace outside of Workspace Context');
  }
  const { updateWorkspaces } = context;

  return useCallback(
    async (id: number): Promise<void> => {
      try {
        await deleteWorkspace({ id });
        updateWorkspaces((prev) =>
          Loadable.map(prev, (ws: Workspace[]) => ws.filter((w) => w.id !== id)),
        );
      } catch (e) {
        handleError(e);
      }
    },
    [updateWorkspaces],
  );
};
