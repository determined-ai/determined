import { useCallback } from 'react';

import { generateContext } from 'contexts';
import { CommandTask } from 'types';

import { getCommands, getNotebooks, getShells, getTensorboards } from '../services/api';

export const Commands = generateContext<CommandTask[] | undefined>({
  initialState: undefined,
  name: 'Commands',
});

export const Notebooks = generateContext<CommandTask[] | undefined>({
  initialState: undefined,
  name: 'Notebooks',
});

export const Shells = generateContext<CommandTask[] | undefined>({
  initialState: undefined,
  name: 'Shells',
});

export const Tensorboards = generateContext<CommandTask[] | undefined>({
  initialState: undefined,
  name: 'Tensorboards',
});

export const useFetchCommands = (canceler: AbortController): () => Promise<void> => {
  const setCommands = Commands.useActionContext();

  return useCallback(async (): Promise<void> => {
    const commandsResponse = await getCommands({ signal: canceler.signal });
    setCommands({ type: Commands.ActionType.Set, value: commandsResponse });
  }, [ canceler, setCommands ]);
};

export const useFetchNotebooks = (canceler: AbortController): () => Promise<void> => {
  const setNotebooks = Notebooks.useActionContext();

  return useCallback(async (): Promise<void> => {
    const notebooksResponse = await getNotebooks({ signal: canceler.signal });
    setNotebooks({ type: Commands.ActionType.Set, value: notebooksResponse });
  }, [ canceler, setNotebooks ]);
};

export const useFetchShells = (canceler: AbortController): () => Promise<void> => {
  const setShells = Shells.useActionContext();

  return useCallback(async (): Promise<void> => {
    const shellsResponse = await getShells({ signal: canceler.signal });
    setShells({ type: Commands.ActionType.Set, value: shellsResponse });
  }, [ canceler, setShells ]);
};

export const useFetchTensorboards = (canceler: AbortController): () => Promise<void> => {
  const setTensorboards = Tensorboards.useActionContext();

  return useCallback(async (): Promise<void> => {
    try {
      const tensorboardsResponse = await getTensorboards({ signal: canceler.signal });
      setTensorboards({ type: Commands.ActionType.Set, value: tensorboardsResponse });
    } catch (e) {}
  }, [ canceler, setTensorboards ]);
};
