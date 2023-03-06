import { observable, WritableObservable } from 'micro-observables';
import React, { createContext, PropsWithChildren, useCallback, useContext, useRef } from 'react';

import { createWorkspace, deleteWorkspace, getWorkspaces } from 'services/api';
import { V1PostWorkspaceRequest } from 'services/api-ts-sdk';
import { GetWorkspacesParams } from 'services/types';
import { isEqual } from 'shared/utils/data';
import { Workspace } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';
import { useValueMemoizedObservable } from 'utils/observable';

type WorkspacesContext = { workspaces: WritableObservable<Loadable<Workspace[]>> };

const WorkspacesContext = createContext<WorkspacesContext | null>(null);

export const WorkspacesProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const workspaces = useRef<WritableObservable<Loadable<Workspace[]>>>(observable(NotLoaded));

  return (
    <WorkspacesContext.Provider value={{ workspaces: workspaces.current }}>
      {children}
    </WorkspacesContext.Provider>
  );
};

export const useFetchWorkspaces = (
  canceler: AbortController,
): ((settings?: GetWorkspacesParams) => Promise<void>) => {
  const context = useContext(WorkspacesContext);

  if (context === null) {
    throw new Error('Attempted to use useFetchWorkspaces outside of Workspace Context');
  }

  const { workspaces } = context;

  return useCallback(
    async (settings = {} as GetWorkspacesParams): Promise<void> => {
      try {
        const response = await getWorkspaces(settings, { signal: canceler.signal });

        workspaces.set(Loaded(response.workspaces));
      } catch (e) {
        handleError(e);
      }
    },
    [canceler, workspaces],
  );
};

export const useEnsureWorkspacesFetched = (
  canceler: AbortController,
): ((settings?: GetWorkspacesParams) => Promise<void>) => {
  const context = useContext(WorkspacesContext);

  if (context === null) {
    throw new Error('Attempted to use useEnsureWorkspacesFetched outside of Workspace Context');
  }

  const { workspaces } = context;
  const memoWorkspaces = useValueMemoizedObservable(workspaces);

  return useCallback(
    async (settings = {} as GetWorkspacesParams): Promise<void> => {
      if (memoWorkspaces !== NotLoaded) return;

      try {
        const response = await getWorkspaces(settings, { signal: canceler.signal });

        if (!isEqual(memoWorkspaces, response.workspaces))
          workspaces.set(Loaded(response.workspaces));
      } catch (e) {
        handleError(e);
      }
    },
    [canceler, workspaces, memoWorkspaces],
  );
};

export const useWorkspaces = (params?: GetWorkspacesParams): Loadable<Workspace[]> => {
  const context = useContext(WorkspacesContext);

  if (context === null) {
    throw new Error('Attempted to use useWorkspaces outside of Workspace Context');
  }

  const { workspaces } = context;
  const workspacesState = useValueMemoizedObservable(workspaces);

  return Loadable.map(workspacesState, (workspaces: Workspace[]) =>
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

  const { workspaces } = context;

  return useCallback(
    (id: number, updater: (arg0: Workspace) => Workspace): void => {
      workspaces.update((ldbWorkspace) => {
        return Loadable.map(ldbWorkspace, (ws) =>
          ws.map((old) => (old.id === id ? updater(old) : old)),
        );
      });
    },
    [workspaces],
  );
};

export const useCreateWorkspace = (): ((arg0: V1PostWorkspaceRequest) => Promise<Workspace>) => {
  const context = useContext(WorkspacesContext);
  if (context === null) {
    throw new Error('Attempted to use useCreateWorkspace outside of Workspace Context');
  }

  const { workspaces } = context;

  return useCallback(
    async (params: V1PostWorkspaceRequest): Promise<Workspace> => {
      const createdWs = await createWorkspace(params);
      workspaces.update((ldbWorkspaces) => {
        return Loadable.map(ldbWorkspaces, (ws: Workspace[]) => [...ws, createdWs]);
      });

      return createdWs;
    },
    [workspaces],
  );
};

// On logout, clear old workspace records.
export const useResetWorkspaces = (): (() => void) => {
  const context = useContext(WorkspacesContext);

  if (context === null) {
    throw new Error('Attempted to use useResetWorkspaces outside of Workspace Context');
  }

  const { workspaces } = context;

  return useCallback((): void => {
    workspaces.set(NotLoaded);
  }, [workspaces]);
};

export const useDeleteWorkspace = (): ((id: number) => Promise<void>) => {
  const context = useContext(WorkspacesContext);

  if (context === null) {
    throw new Error('Attempted to use useDeleteWorkspace outside of Workspace Context');
  }

  const { workspaces } = context;

  return useCallback(
    async (id: number): Promise<void> => {
      try {
        await deleteWorkspace({ id });

        workspaces.update((ldbWorkspaces) =>
          Loadable.map(ldbWorkspaces, (ws) => ws.filter((w) => w.id !== id)),
        );
      } catch (e) {
        handleError(e);
      }
    },
    [workspaces],
  );
};
