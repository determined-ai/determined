import { useCallback } from 'react';

import { generateContext } from 'contexts';
import { RestApiState } from 'hooks/useRestApi';
import { CommandTask } from 'types';
import { clone } from 'utils/data';

import { getCommands, getNotebooks, getShells, getTensorboards } from '../services/api';

const initialState = {
  errorCount: 0,
  hasLoaded: false,
  isLoading: false,
};

export const Commands = generateContext<RestApiState<CommandTask[]>>({
  initialState: clone(initialState),
  name: 'Commands',
});

export const useFetchCommands = (canceler: AbortController): () => Promise<void> => {
  const setCommands = Commands.useActionContext();

  return useCallback(async (): Promise<void> => {
    const commandsResponse = await getCommands({ signal: canceler.signal });
    setCommands({
      type: Commands.ActionType.Set,
      value: {
        data: commandsResponse,
        errorCount: 0,
        hasLoaded: true,
        isLoading: false,
      },
    });
  }, [ canceler, setCommands ]);
};

export const useFetchNotebooks = (canceler: AbortController): () => Promise<void> => {
  const setNotebooks = Notebooks.useActionContext();

  return useCallback(async (): Promise<void> => {
    const notebooksResponse = await getNotebooks({ signal: canceler.signal });
    setNotebooks({
      type: Commands.ActionType.Set,
      value: {
        data: notebooksResponse,
        errorCount: 0,
        hasLoaded: true,
        isLoading: false,
      },
    });
  }, [ canceler, setNotebooks ]);
};

export const useFetchShells = (canceler: AbortController): () => Promise<void> => {
  const setShells = Shells.useActionContext();

  return useCallback(async (): Promise<void> => {
    const shellsResponse = await getShells({ signal: canceler.signal });
    setShells({
      type: Commands.ActionType.Set,
      value: {
        data: shellsResponse,
        errorCount: 0,
        hasLoaded: true,
        isLoading: false,
      },
    });
  }, [ canceler, setShells ]);
};

export const useFetchTensorboards = (canceler: AbortController): () => Promise<void> => {
  const setTensorboards = Tensorboards.useActionContext();

  return useCallback(async (): Promise<void> => {
    try {
      const tensorboardsResponse = await getTensorboards({ signal: canceler.signal });
      setTensorboards({
        type: Commands.ActionType.Set,
        value: {
          data: tensorboardsResponse,
          errorCount: 0,
          hasLoaded: true,
          isLoading: false,
        },
      });
    } catch (e) {}
  }, [ canceler, setTensorboards ]);
};

export const Notebooks = generateContext<RestApiState<CommandTask[]>>({
  initialState: clone(initialState),
  name: 'Notebooks',
});

export const Shells = generateContext<RestApiState<CommandTask[]>>({
  initialState: clone(initialState),
  name: 'Shells',
});

export const Tensorboards = generateContext<RestApiState<CommandTask[]>>({
  initialState: clone(initialState),
  name: 'Tensorboards',
});
