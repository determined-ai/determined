import { useCallback } from 'react';

import { activeCommandStates, activeRunStates } from 'constants/states';
import { agentsToOverview, StoreAction, useStore, useStoreDispatch } from 'contexts/Store';
import {
  getAgents,
  getCommands,
  getExperiments,
  getInfo,
  getJupyterLabs,
  getResourcePools,
  getShells,
  getTensorBoards,
  getUsers,
  getWorkspaces,
} from 'services/api';
import { ErrorType } from 'shared/utils/error';
import { CommandTask, CommandType, ResourceType } from 'types';
import { updateFaviconType } from 'utils/browser';
import handleError from 'utils/error';

export const useFetchActiveExperiments = (canceler: AbortController): () => Promise<void> => {
  const storeDispatch = useStoreDispatch();

  return useCallback(async (): Promise<void> => {
    try {
      const response = await getExperiments(
        { limit: -2, states: activeRunStates },
        { signal: canceler.signal },
      );
      storeDispatch({
        type: StoreAction.SetActiveExperiments,
        value: response.pagination.total || 0,
      });
    } catch (e) {
      handleError({
        message: 'Unable to fetch active experiments.',
        silent: true,
        type: ErrorType.Api,
      });
    }
  }, [ canceler, storeDispatch ]);
};

export const useFetchAgents = (canceler: AbortController): () => Promise<void> => {
  const { info } = useStore();
  const storeDispatch = useStoreDispatch();

  return useCallback(async (): Promise<void> => {
    try {
      const response = await getAgents({ signal: canceler.signal });
      const cluster = agentsToOverview(response);
      storeDispatch({ type: StoreAction.SetAgents, value: response });
      updateFaviconType(cluster[ResourceType.ALL].allocation !== 0, info.branding);
    } catch (e) { handleError(e); }
  }, [ canceler, info.branding, storeDispatch ]);
};

export const useFetchInfo = (canceler: AbortController): () => Promise<void> => {
  const storeDispatch = useStoreDispatch();

  return useCallback(async (): Promise<void> => {
    try {
      const response = await getInfo({ signal: canceler.signal });
      storeDispatch({ type: StoreAction.SetInfo, value: response });
    } catch (e) {
      storeDispatch({ type: StoreAction.SetInfoCheck });
      handleError(e);
    }
  }, [ canceler, storeDispatch ]);
};

export const useFetchUsers = (canceler: AbortController): () => Promise<void> => {
  const storeDispatch = useStoreDispatch();

  return useCallback(async (): Promise<void> => {
    try {
      const usersResponse = await getUsers({ signal: canceler.signal });
      storeDispatch({ type: StoreAction.SetUsers, value: usersResponse });
    } catch (e) { handleError(e); }
  }, [ canceler, storeDispatch ]);
};

export const useFetchResourcePools = (canceler: AbortController): () => Promise<void> => {
  const storeDispatch = useStoreDispatch();
  return useCallback(async (): Promise<void> => {
    try {
      const resourcePools = await getResourcePools({}, { signal: canceler.signal });
      storeDispatch({ type: StoreAction.SetResourcePools, value: resourcePools });
    } catch (e) { handleError(e); }
  }, [ canceler, storeDispatch ]);
};

export const useFetchTasks = (canceler: AbortController): () => Promise<void> => {
  const storeDispatch = useStoreDispatch();

  const countActiveCommand = (commands: CommandTask[]): number => {
    return commands.filter(command => activeCommandStates.includes(command.state)).length;
  };

  return useCallback(async (): Promise<void> => {
    try {
      const [ commands, notebooks, shells, tensorboards ] = await Promise.all([
        getCommands({ signal: canceler.signal }),
        getJupyterLabs({ signal: canceler.signal }),
        getShells({ signal: canceler.signal }),
        getTensorBoards({ signal: canceler.signal }),
      ]);
      const combined = {
        [CommandType.Command]: countActiveCommand(commands),
        [CommandType.JupyterLab]: countActiveCommand(notebooks),
        [CommandType.Shell]: countActiveCommand(shells),
        [CommandType.TensorBoard]: countActiveCommand(tensorboards),
      };
      storeDispatch({ type: StoreAction.SetTasks, value: combined });
    } catch (e) {
      handleError({ message: 'Unable to fetch tasks.', silent: true, type: ErrorType.Api });
    }
  }, [ canceler, storeDispatch ]);
};

export const useFetchPinnedWorkspaces = (canceler: AbortController): () => Promise<void> => {
  const storeDispatch = useStoreDispatch();
  return useCallback(async (): Promise<void> => {
    try {
      const pinnedWorkspaces = await getWorkspaces(
        { limit: 0, pinned: true },
        { signal: canceler.signal },
      );
      storeDispatch({ type: StoreAction.SetPinnedWorkspaces, value: pinnedWorkspaces.workspaces });
    } catch (e) { handleError(e); }
  }, [ canceler, storeDispatch ]);
};
