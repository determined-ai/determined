import { Dispatch } from 'react';

import { Action, State } from 'contexts/Store';

interface ES {
  dispatch: Dispatch<Action>,
  state: Partial<State>
}

export let store: null | ES = null;

export const exposeStore = (x: ES): void => {
  store = x;
};
