import { theme as AntdTheme, ConfigProvider } from 'antd';
import React, { Dispatch, useContext, useLayoutEffect, useMemo, useReducer, useRef } from 'react';

import { RecordKey } from 'components/kit/internal/types';
import { themeLightDetermined } from 'components/kit/Theme';

import { DarkLight, globalCssVars, Mode, Theme } from './themeUtils';

interface StateUI {
  chromeCollapsed: boolean;
  darkLight: DarkLight;
  isPageHidden: boolean;
  mode: Mode;
  showChrome: boolean;
  showSpinner: boolean;
  theme: Theme;
}

const initUI: StateUI = {
  chromeCollapsed: false,
  darkLight: DarkLight.Light,
  isPageHidden: false,
  mode: Mode.System,
  showChrome: true,
  showSpinner: false,
  theme: {} as Theme,
};

const StoreActionUI = {
  HideUIChrome: 'HideUIChrome',
  HideUISpinner: 'HideUISpinner',
  SetMode: 'SetMode',
  SetPageVisibility: 'SetPageVisibility',
  SetTheme: 'SetTheme',
  ShowUIChrome: 'ShowUIChrome',
  ShowUISpinner: 'ShowUISpinner',
} as const;

type ActionUI =
  | { type: typeof StoreActionUI.HideUIChrome }
  | { type: typeof StoreActionUI.HideUISpinner }
  | { type: typeof StoreActionUI.SetMode; value: Mode }
  | { type: typeof StoreActionUI.SetPageVisibility; value: boolean }
  | { type: typeof StoreActionUI.SetTheme; value: { darkLight: DarkLight; theme: Theme } }
  | { type: typeof StoreActionUI.ShowUIChrome }
  | { type: typeof StoreActionUI.ShowUISpinner };

class UIActions {
  constructor(private dispatch: Dispatch<ActionUI>) {}

  public hideChrome = (): void => {
    this.dispatch({ type: StoreActionUI.HideUIChrome });
  };

  public hideSpinner = (): void => {
    this.dispatch({ type: StoreActionUI.HideUISpinner });
  };

  public setMode = (mode: Mode): void => {
    this.dispatch({ type: StoreActionUI.SetMode, value: mode });
  };

  public setPageVisibility = (isPageHidden: boolean): void => {
    this.dispatch({ type: StoreActionUI.SetPageVisibility, value: isPageHidden });
  };

  public setTheme = (darkLight: DarkLight, theme: Theme): void => {
    this.dispatch({ type: StoreActionUI.SetTheme, value: { darkLight, theme } });
  };

  public showChrome = (): void => {
    this.dispatch({ type: StoreActionUI.ShowUIChrome });
  };

  public showSpinner = (): void => {
    this.dispatch({ type: StoreActionUI.ShowUISpinner });
  };
}

const camelCaseToKebab = (text: string): string => {
  return text
    .trim()
    .split('')
    .map((char, index) => {
      return char === char.toUpperCase() ? `${index !== 0 ? '-' : ''}${char.toLowerCase()}` : char;
    })
    .join('');
};

/**
 * return a part of the input state that should be updated.
 * @param state ui state
 * @param action
 * @returns
 */
const reducerUI = (state: StateUI, action: ActionUI): Partial<StateUI> | void => {
  switch (action.type) {
    case StoreActionUI.HideUIChrome:
      if (!state.showChrome) return;
      return { showChrome: false };
    case StoreActionUI.HideUISpinner:
      if (!state.showSpinner) return;
      return { showSpinner: false };
    case StoreActionUI.SetMode:
      return { mode: action.value };
    case StoreActionUI.SetPageVisibility:
      return { isPageHidden: action.value };
    case StoreActionUI.SetTheme:
      return {
        darkLight: action.value.darkLight,
        theme: action.value.theme,
      };
    case StoreActionUI.ShowUIChrome:
      if (state.showChrome) return;
      return { showChrome: true };
    case StoreActionUI.ShowUISpinner:
      if (state.showSpinner) return;
      return { showSpinner: true };
    default:
      return;
  }
};
const StateContext = React.createContext<StateUI | undefined>(undefined);
const DispatchContext = React.createContext<Dispatch<ActionUI> | undefined>(undefined);

const reducer = (state: StateUI, action: ActionUI): StateUI => {
  const newState = reducerUI(state, action);
  return { ...state, ...newState }; // TODO: check for deep equality here instead of on the full state
};

const useUI = (): { actions: UIActions; ui: StateUI } => {
  const context = useContext(StateContext);
  if (context === undefined) {
    throw new Error('useStore(UI) must be used within a UIProvider');
  }
  const dispatchContext = useContext(DispatchContext);
  if (dispatchContext === undefined) {
    throw new Error('useStoreDispatch must be used within a UIProvider');
  }
  const uiActions = useMemo(() => new UIActions(dispatchContext), [dispatchContext]);
  return { actions: uiActions, ui: context };
};

export const ThemeProvider: React.FC<{
  children?: React.ReactNode;
}> = ({ children }) => {
  const [state, dispatch] = useReducer(reducer, initUI);
  // const systemMode = getSystemMode();
  // const userTheme = userSettings.get
  const theme = themeLightDetermined;

  // Update darkLight and theme when branding, system mode, or mode changes.
  useLayoutEffect(() => {
    const darkLight = DarkLight.Light;
    dispatch({
      type: StoreActionUI.SetTheme,
      value: { darkLight, theme },
    });
  }, [state.mode, theme]);

  return (
    <StateContext.Provider value={state}>
      <DispatchContext.Provider value={dispatch}>{children}</DispatchContext.Provider>
    </StateContext.Provider>
  );
};

export const UIProvider: React.FC<{
  children?: React.ReactNode;
  theme: Theme;
  darkMode: boolean;
}> = ({ children, theme, darkMode = false }) => {
  const ref = useRef<HTMLDivElement>(null);

  // Set global CSS variables shared across themes.
  Object.keys(globalCssVars).forEach((key) => {
    const value = (globalCssVars as Record<RecordKey, string>)[key];
    ref.current?.style.setProperty(`--${camelCaseToKebab(key)}`, value);
  });

  // Set each theme property as top level CSS variable.
  Object.keys(theme).forEach((key) => {
    const value = (theme as Record<RecordKey, string>)[key];
    ref.current?.style.setProperty(`--theme-${camelCaseToKebab(key)}`, value);
  });

  const configTheme = {
    algorithm: darkMode ? AntdTheme.darkAlgorithm : AntdTheme.defaultAlgorithm,
    components: {
      Button: {
        colorBgContainer: 'transparent',
      },
      Progress: {
        marginXS: 0,
      },
      Tooltip: {
        colorBgDefault: 'var(--theme-float)',
        colorTextLightSolid: 'var(--theme-float-on)',
      },
    },
    token: {
      borderRadius: 2,
      colorPrimary: '#1890ff',
      fontFamily: 'var(--theme-font-family)',
    },
  };

  return (
    <div className="ui-provider" ref={ref}>
      <ConfigProvider theme={configTheme}>{children}</ConfigProvider>
    </div>
  );
};

export default useUI;
