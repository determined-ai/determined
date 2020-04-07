import { generateContext } from 'contexts';
import { RestApiState } from 'hooks/useRestApi';
import { Command } from 'types';
import { clone } from 'utils/data';

const initialState = {
  errorCount: 0,
  hasLoaded: false,
  isLoading: false,
};

export const Commands = generateContext<RestApiState<Command[]>>({
  initialState: clone(initialState),
  name: 'Commands',
});

export const Notebooks = generateContext<RestApiState<Command[]>>({
  initialState: clone(initialState),
  name: 'Notebooks',
});

export const Shells = generateContext<RestApiState<Command[]>>({
  initialState: clone(initialState),
  name: 'Shells',
});

export const Tensorboards = generateContext<RestApiState<Command[]>>({
  initialState: clone(initialState),
  name: 'Tensorboards',
});
