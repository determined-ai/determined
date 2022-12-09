import { Map } from 'immutable';
import React, { createContext, PropsWithChildren, useCallback, useContext, useState } from 'react';

import { getWorkspaces } from 'services/api';
import { GetWorkspacesParams } from 'services/types';
import { Workspace } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';
import { encodeParams } from 'utils/store';

type WorkspacesContext = {
  updateWorkspaces: (ws: Map<number, Workspace>) => void;
  updateWorkspacesByParams: (ws: Map<string, number[]>) => void;
  workspaces: Map<number, Workspace>;
  workspacesByParams: Map<string, number[]>;
};

const WorkspacesContext = createContext<WorkspacesContext | null>(null);

export const WorkspacesProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const [workspaces, updateWorkspaces] = useState<Map<number, Workspace>>(Map<number, Workspace>());
  const [workspacesByParams, updateWorkspacesByParams] = useState<Map<string, number[]>>(
    Map<string, number[]>(),
  );
  return (
    <WorkspacesContext.Provider
      value={{ updateWorkspaces, updateWorkspacesByParams, workspaces, workspacesByParams }}>
      {children}
    </WorkspacesContext.Provider>
  );
};

const filterWorkspaces = (workspaces: Workspace[], params: GetWorkspacesParams): Workspace[] =>
  workspaces.filter((ws) =>
    Object.keys(params).reduce<boolean>((accumulator: boolean, key) => {
      switch (key) {
        case 'archived':
          return accumulator && ws.archived === params.archived;
        case 'pinned':
          return accumulator && ws.pinned === params.pinned;
        case 'name':
          return accumulator && ws.name === params.name;
        case 'users':
          return accumulator && (params.users || []).indexOf(String(ws.userId)) > -1;
        default:
          return false;
      }
    }, true),
  );

export const useFetchWorkspaces = (
  params: GetWorkspacesParams,
  canceler: AbortController,
): (() => Promise<void>) => {
  const context = useContext(WorkspacesContext);
  if (context === null) {
    throw new Error('Attempted to use useFetchWorkspaces outside of Workspace Context');
  }
  const { updateWorkspaces, updateWorkspacesByParams, workspacesByParams } = context;

  return useCallback(async (): Promise<void> => {
    try {
      const response = await getWorkspaces({}, { signal: canceler.signal });
      updateWorkspaces(Map<number, Workspace>(response.workspaces.map((ws) => [ws.id, ws])));

      const filterIds = filterWorkspaces(response.workspaces, params).map((ws) => ws.id);
      updateWorkspacesByParams(workspacesByParams.set(encodeParams(params), filterIds));
    } catch (e) {
      handleError(e);
    }
  }, [canceler, updateWorkspaces]);
};

export const useEnsureWorkspacesFetched = (
  params: GetWorkspacesParams,
  canceler: AbortController,
): (() => Promise<void>) => {
  const context = useContext(WorkspacesContext);
  if (context === null) {
    throw new Error('Attempted to use useEnsureWorkspacesFetched outside of Workspace Context');
  }
  const { workspaces, updateWorkspaces, updateWorkspacesByParams, workspacesByParams } = context;

  return useCallback(async (): Promise<void> => {
    if (workspaces.size !== 0) return;
    try {
      const response = await getWorkspaces({}, { signal: canceler.signal });
      updateWorkspaces(Map<number, Workspace>(response.workspaces.map((ws) => [ws.id, ws])));

      const filterIds = filterWorkspaces(response.workspaces, params).map((ws) => ws.id);
      updateWorkspacesByParams(workspacesByParams.set(encodeParams(params), filterIds));
    } catch (e) {
      handleError(e);
    }
  }, [canceler, workspaces]);
};

export const useWorkspaces = (params: GetWorkspacesParams): Loadable<Workspace[]> => {
  const context = useContext(WorkspacesContext);
  if (context === null) {
    throw new Error('Attempted to use useWorkspaces outside of Workspace Context');
  }
  const matchingIds = context.workspacesByParams.get(encodeParams(params));
  const val = Array.from(context.workspaces.values()).filter((ws) => matchingIds?.includes(ws.id));

  return val ? Loaded(val) : NotLoaded;
};
