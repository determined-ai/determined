import { generateContext } from 'contexts';

enum ActionType {
  HideChrome,
  HideSpinner,
  ShowSpinner,
}

type State = {
  showChrome: boolean;
  showSpinner: boolean;
}

type Action =
  | { type: ActionType.HideChrome }
  | { type: ActionType.HideSpinner }
  | { type: ActionType.ShowSpinner }

const defaultState = {
  showChrome: true,
  showSpinner: false,
};

const reducer = (state: State, action: Action): State => {
  switch (action.type) {
    case ActionType.HideChrome:
      return { ...state, showChrome: false };
    case ActionType.HideSpinner:
      if (!state.showSpinner) return state;
      return { ...state, showSpinner: false };
    case ActionType.ShowSpinner:
      if (state.showSpinner) return state;
      return { ...state, showSpinner: true };
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
