import { generateContext } from 'contexts';

enum ActionType {
  Hide,
  Show,
}

type State = {
  isOpaque: boolean;
  isShowing: boolean;
}

type Action =
  | { type: ActionType.Hide }
  | { type: ActionType.Show; opaque?: boolean }

const defaultState = {
  isOpaque: false,
  isShowing: false,
};

const reducer = (state: State, action: Action): State => {
  switch (action.type) {
    case ActionType.Hide:
      if (!state.isShowing) return state;
      return { isOpaque: state.isOpaque, isShowing: false };
    case ActionType.Show:
      if (state.isShowing) return state;
      return { isOpaque: action.opaque != null ? action.opaque : state.isOpaque, isShowing: true };
    default:
      return state;
  }
};

const contextProvider = generateContext<State, Action>({
  initialState: defaultState,
  name: 'Spinner',
  reducer,
});

export default { ...contextProvider, ActionType };
