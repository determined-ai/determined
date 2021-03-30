import React, { Dispatch, useContext, useReducer } from 'react';

import { clone } from 'utils/data';

enum ActionType { Reset, Set }

interface Options<T, A> {
  initialState: T;
  name: string;
  reducer?: (state: T, action: A) => T;
}

type Action<T> =
| { type: ActionType.Reset; value: T }
| { type: ActionType.Set; value: T }

type Export<T, A> = {
  ActionType: typeof ActionType;
  Provider: React.FC;
  useActionContext: () => Dispatch<A>;
  useStateContext: () => T;
};

const generateContextHook = <T>(
  contextName: string,
  hookName: string,
  context: React.Context<T | null>,
) => {
  return (): T => {
    const ctx = useContext<T | null>(context);
    if (ctx !== null) return ctx;
    throw new Error(`${hookName} must be used inside the ${contextName}.Provider.`);
  };
};

export const generateContext = <T, A = Action<T>>(options: Options<T, A>): Export<T, A> => {
  const initialState = clone(options.initialState);
  const StateContext = React.createContext<T | null>(null);
  const ActionContext = React.createContext<Dispatch<A> | null>(null);

  const defaultReducer = (state: T, action: Action<T>): T => {
    switch (action.type) {
      case ActionType.Reset:
        return initialState;
      case ActionType.Set:
        return clone(action.value);
      default:
        return state;
    }
  };
  const reducer = options.reducer || defaultReducer;

  interface Props {
    children?: React.ReactNode
  }

  const Provider: React.FC<Props> = (props: Props) => {
    const [ state, dispatch ] = useReducer(reducer, initialState);

    return React.createElement(
      StateContext.Provider,
      { value: state },
      React.createElement(
        ActionContext.Provider,
        { value: dispatch },
        props.children,
      ),
    );
  };

  return {
    ActionType,
    Provider,
    useActionContext: generateContextHook<Dispatch<A>>(
      options.name,
      'useActionContext',
      ActionContext,
    ),
    useStateContext: generateContextHook<T>(
      options.name,
      'useStateContext',
      StateContext,
    ),
  };
};
