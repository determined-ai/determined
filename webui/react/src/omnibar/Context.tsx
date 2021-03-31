import { generateContext } from 'contexts';

enum ActionType {
  Hide,
  Show,
}

type State = {
  isShowing: boolean;
}

type Action =
  | { type: ActionType.Hide }
  | { type: ActionType.Show; opaque?: boolean }

const defaultState = { isShowing: false };

const reducer = (state: State, action: Action): State => {
  switch (action.type) {
    case ActionType.Hide:
      return { isShowing: false };
    case ActionType.Show:
      return { isShowing: true };
    default:
      return state;
  }
};

const contextProvider = generateContext<State, Action>({
  initialState: defaultState,
  name: 'Omnibar',
  reducer,
});

export default { ...contextProvider, ActionType };
