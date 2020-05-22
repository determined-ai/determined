import { generateContext } from 'contexts';

enum ActionType {
  Hide,
  Show,
}

type Action =
  | { type: ActionType.Hide }
  | { type: ActionType.Show }

const defaultState = false;

const reducer = (state: boolean, action: Action): boolean => {
  switch (action.type) {
    case ActionType.Hide:
      return false;
    case ActionType.Show:
      return true;
    default:
      return state;
  }
};

const contextProvider = generateContext<boolean, Action>({
  initialState: defaultState,
  name: 'Spinner',
  reducer,
});

export default { ...contextProvider, ActionType };
