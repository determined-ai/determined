import { Theme } from 'hew/Theme';
import { DefaultTheme } from 'hew/Theme/themes';
import _ from 'lodash';
import { useObservable } from 'micro-observables';
import React, {
  Dispatch,
  useCallback,
  useContext,
  useEffect,
  useLayoutEffect,
  useMemo,
  useReducer,
  useState,
} from 'react';

import determinedInfo from 'stores/determinedInfo';
import { ValueOf } from 'types';
export const Mode = {
  Dark: 'dark',
  Light: 'light',
  System: 'system',
} as const;

export type Mode = ValueOf<typeof Mode>;
type DarkLight = ValueOf<typeof DarkLight>;
/**
 * DarkLight is a resolved form of `Mode` where we figure out
 * what `Mode.System` should ultimate resolve to (`Dark` vs `Light).
 */
const DarkLight = {
  Dark: 'dark',
  Light: 'light',
} as const;

export const BrandingType = {
  Determined: 'determined',
  HPE: 'hpe',
} as const;

export type BrandingType = ValueOf<typeof BrandingType>;

export const MATCH_MEDIA_SCHEME_DARK = '(prefers-color-scheme: dark)';
export const MATCH_MEDIA_SCHEME_LIGHT = '(prefers-color-scheme: light)';

const themes = {
  [BrandingType.Determined]: {
    [DarkLight.Dark]: DefaultTheme.Dark,
    [DarkLight.Light]: DefaultTheme.Light,
  },
  [BrandingType.HPE]: {
    [DarkLight.Dark]: DefaultTheme.HPEDark,
    [DarkLight.Light]: DefaultTheme.HPELight,
  },
};

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

const getDarkLight = (mode: Mode, systemMode: Mode): DarkLight => {
  const resolvedMode =
    mode === Mode.System ? (systemMode === Mode.System ? Mode.Light : systemMode) : mode;
  return resolvedMode === Mode.Light ? DarkLight.Light : DarkLight.Dark;
};

export const getSystemMode = (): Mode => {
  const isDark = matchMedia?.(MATCH_MEDIA_SCHEME_DARK).matches;
  if (isDark) return Mode.Dark;

  const isLight = matchMedia?.(MATCH_MEDIA_SCHEME_LIGHT).matches;
  if (isLight) return Mode.Light;

  return Mode.System;
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

const useUI = (): { actions: UIActions; ui: StateUI; theme: Theme; isDarkMode: boolean } => {
  const context = useContext(StateContext);
  if (context === undefined) {
    throw new Error('useStore(UI) must be used within a ThemeProvider');
  }
  const dispatchContext = useContext(DispatchContext);
  if (dispatchContext === undefined) {
    throw new Error('useStoreDispatch must be used within a ThemeProvider');
  }
  const uiActions = useMemo(() => new UIActions(dispatchContext), [dispatchContext]);

  const info = useObservable(determinedInfo.info);
  const mode = context.mode;
  const branding = info?.branding || BrandingType.Determined;
  const [systemMode, setSystemMode] = useState<Mode>(() => getSystemMode());

  const darkLight = getDarkLight(mode, systemMode);

  const theme = useMemo(() => {
    const userTheme = context.theme;
    return userTheme && !_.isEqual(userTheme, {}) ? userTheme : themes[branding][darkLight];
  }, [context, branding, darkLight]);

  const isDarkMode = useMemo(() => darkLight === DarkLight.Dark, [darkLight]);

  const handleSchemeChange = useCallback((event: MediaQueryListEvent) => {
    if (!event.matches) setSystemMode(getSystemMode());
  }, []);

  // Detect browser/OS level dark/light mode changes.
  useEffect(() => {
    matchMedia?.(MATCH_MEDIA_SCHEME_DARK).addEventListener('change', handleSchemeChange);
    matchMedia?.(MATCH_MEDIA_SCHEME_LIGHT).addEventListener('change', handleSchemeChange);

    return () => {
      matchMedia?.(MATCH_MEDIA_SCHEME_DARK).removeEventListener('change', handleSchemeChange);
      matchMedia?.(MATCH_MEDIA_SCHEME_LIGHT).removeEventListener('change', handleSchemeChange);
    };
  }, [handleSchemeChange]);

  return { actions: uiActions, isDarkMode, theme, ui: context };
};

export const ThemeProvider: React.FC<{ children?: React.ReactNode; branding?: BrandingType }> = ({
  children,
  branding,
}) => {
  const [state, dispatch] = useReducer(reducer, initUI);
  const [systemMode, setSystemMode] = useState<Mode>(() => getSystemMode());

  const handleSchemeChange = useCallback((event: MediaQueryListEvent) => {
    if (!event.matches) setSystemMode(getSystemMode());
  }, []);

  // Detect browser/OS level dark/light mode changes.
  useEffect(() => {
    matchMedia?.(MATCH_MEDIA_SCHEME_DARK).addEventListener('change', handleSchemeChange);
    matchMedia?.(MATCH_MEDIA_SCHEME_LIGHT).addEventListener('change', handleSchemeChange);

    return () => {
      matchMedia?.(MATCH_MEDIA_SCHEME_DARK).removeEventListener('change', handleSchemeChange);
      matchMedia?.(MATCH_MEDIA_SCHEME_LIGHT).removeEventListener('change', handleSchemeChange);
    };
  }, [handleSchemeChange]);

  // Update darkLight and theme when branding, system mode, or mode changes.
  useLayoutEffect(() => {
    const darkLight = getDarkLight(state.mode, systemMode);

    dispatch({
      type: StoreActionUI.SetTheme,
      value: { darkLight, theme: themes[branding || 'determined'][darkLight] },
    });
  }, [branding, systemMode, state.mode]);

  return (
    <StateContext.Provider value={state}>
      <DispatchContext.Provider value={dispatch}>{children}</DispatchContext.Provider>
    </StateContext.Provider>
  );
};

export default useUI;
