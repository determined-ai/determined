import { generateContext } from 'contexts';

enum ActionType {
  HideChrome,
  HideSpinner,
  ShowSpinner,
}

type State = {
  opaqueSpinner: boolean;
  showChrome: boolean;
  showSpinner: boolean;
}

type Action =
  | { type: ActionType.HideChrome }
  | { type: ActionType.HideSpinner }
  | { type: ActionType.ShowSpinner; opaque?: boolean }

const defaultState = {
  opaqueSpinner: false,
  showChrome: true,
  showSpinner: false,
};

const reducer = (state: State, action: Action): State => {
  switch (action.type) {
    case ActionType.HideChrome:
      return { ...state, showChrome: false };
    case ActionType.HideSpinner:
      if (!state.showSpinner) return state;
      return { ...state, opaqueSpinner: state.opaqueSpinner, showSpinner: false };
    case ActionType.ShowSpinner:
      if (state.showSpinner) return state;
      return {
        ...state,
        opaqueSpinner: action.opaque != null ? action.opaque : state.opaqueSpinner,
        showSpinner: true,
      };
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
