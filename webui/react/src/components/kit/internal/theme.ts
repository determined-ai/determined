/* eslint-disable sort-keys-fix/sort-keys-fix */

import React, { Dispatch, useContext, useMemo } from 'react';

import { isColor, rgba2str, rgbaMix, str2rgba } from 'components/kit/internal/color';

const STRONG_WEAK_DELTA = 45;

export type Theme = Record<keyof typeof themeBase, string>;

const ThemeStateAction = {
  SetDarkMode: 'SetDarkMode',
} as const;

type ThemeStateAction = { type: typeof ThemeStateAction.SetDarkMode; value: boolean };

class ThemeStateActions {
  constructor(private dispatch: Dispatch<ThemeStateAction>) {}

  public setDarkMode = (isDarkMode: boolean): void => {
    this.dispatch({ type: ThemeStateAction.SetDarkMode, value: isDarkMode });
  };
}

interface ThemeState {
  darkMode: boolean;
}

const reducerThemeState = (action: ThemeStateAction): Partial<ThemeState> | void => {
  switch (action.type) {
    case ThemeStateAction.SetDarkMode:
      return { darkMode: action.value };
    default:
      return;
  }
};

export const ThemeContext = React.createContext<ThemeState | undefined>(undefined);
export const ThemeDispatchContext = React.createContext<Dispatch<ThemeStateAction> | undefined>(
  undefined,
);

export const themeStateReducer = (state: ThemeState, action: ThemeStateAction): ThemeState => {
  const newState = reducerThemeState(action);
  return { ...state, ...newState };
};

export const useThemeState = (): { actions: ThemeStateActions; themeState: ThemeState } => {
  /**
   * Some UI Kit components such as the CodeEditor do not inherit the theme from css or page styling
   * and instead require us to set a theme realted prop dynamically. This context allows us to
   * subscribe to UIProvider theme updates and re-render these child components with the correct
   * thehe.
   */
  const context = useContext(ThemeContext);
  if (context === undefined) {
    throw new Error('useStore(UI) must be used within a UIProvider');
  }
  const dispatchContext = useContext(ThemeDispatchContext);
  if (dispatchContext === undefined) {
    throw new Error('useStoreDispatch must be used within a UIProvider');
  }
  const actions = useMemo(() => new ThemeStateActions(dispatchContext), [dispatchContext]);
  return { actions, themeState: context };
};

const generateStrongWeak = (theme: Theme): Theme => {
  const rgbaStrong = str2rgba(theme.strong);
  const rgbaWeak = str2rgba(theme.weak);

  for (const [key, value] of Object.entries(theme)) {
    const matches = key.match(/^(.+)(Strong|Weak)$/);
    if (matches?.length === 3 && value === undefined) {
      const isStrong = matches[2] === 'Strong';
      const baseKey = matches[1] as keyof Theme;
      const baseValue = theme[baseKey];
      if (baseValue && isColor(baseValue)) {
        const rgba = str2rgba(baseValue);
        const mixer = isStrong ? rgbaStrong : rgbaWeak;
        theme[key as keyof Theme] = rgba2str(rgbaMix(rgba, mixer, STRONG_WEAK_DELTA));
      }
    }
  }
  return theme as Theme;
};

export const themeBase = {
  // Area and surface styles.
  background: undefined,
  backgroundBorder: undefined,
  backgroundBorderStrong: undefined,
  backgroundBorderWeak: undefined,
  backgroundOn: undefined,
  backgroundOnStrong: undefined,
  backgroundOnWeak: undefined,
  backgroundStrong: undefined,
  backgroundWeak: undefined,

  // Brand colors.
  brand: 'rgba(247, 123, 33, 1.0)',
  brandStrong: undefined,
  brandWeak: undefined,

  // Color schemes
  colorScheme: 'normal',
  float: undefined,
  floatBorder: undefined,
  floatBorderStrong: undefined,
  floatBorderWeak: undefined,
  floatOn: undefined,
  floatOnStrong: undefined,
  floatOnWeak: undefined,
  floatStrong: undefined,
  floatWeak: undefined,

  // Font styles.
  fontFamily: 'Inter, Arial, Helvetica, sans-serif',
  fontFamilyCode: '"Source Code Pro", monospace',

  // Specialized and unique styles.
  density: '2',
  fontFamilyVar: '"Inter var", Arial, Helvetica, sans-serif',
  borderRadius: '4px',

  // Interactive styles.
  ix: undefined,
  borderRadiusStrong: '8px',
  ixActive: undefined,
  borderRadiusWeak: '2px',
  ixBorder: undefined,
  elevation: undefined,
  ixBorderActive: undefined,
  elevationStrong: undefined,
  stage: undefined,
  elevationWeak: undefined,
  stageBorder: undefined,
  ixBorderInactive: undefined,
  stageBorderStrong: undefined,
  ixBorderStrong: undefined,
  stageBorderWeak: undefined,
  ixBorderWeak: undefined,

  // Palette colors for strong/weak calculations.
  strong: undefined,
  ixInactive: undefined,
  weak: undefined,
  ixOn: undefined,
  stageOn: undefined,
  ixOnActive: undefined,
  stageOnStrong: undefined,
  ixOnInactive: undefined,
  stageOnWeak: undefined,
  ixOnStrong: undefined,
  stageStrong: undefined,
  ixOnWeak: undefined,
  stageWeak: undefined,
  ixStrong: undefined,
  surface: undefined,
  ixWeak: undefined,
  surfaceBorder: undefined,
  overlay: undefined,
  surfaceBorderStrong: undefined,
  overlayStrong: undefined,
  surfaceOn: undefined,
  overlayWeak: undefined,
  surfaceOnStrong: undefined,

  // Status styles.
  statusActive: 'rgba(0, 155, 222, 1.0)',
  surfaceStrong: undefined,
  statusActiveOn: 'rgba(255, 255, 255, 1.0)',
  surfaceWeak: undefined,
  statusActiveOnStrong: undefined,
  statusActiveOnWeak: undefined,
  surfaceOnWeak: undefined,
  statusActiveStrong: undefined,
  statusActiveWeak: undefined,
  surfaceBorderWeak: undefined,
  statusCritical: 'rgba(204, 0, 0, 1.0)',
  statusCriticalOn: 'rgba(255, 255, 255, 1.0)',
  statusCriticalOnStrong: undefined,
  statusCriticalOnWeak: undefined,
  statusCriticalStrong: undefined,
  statusCriticalWeak: undefined,
  statusError: 'rgb(247, 140, 140)',
  statusInactive: 'rgba(102, 102, 102, 1.0)',
  statusInactiveOn: 'rgba(255, 255, 255, 1.0)',
  statusInactiveOnStrong: undefined,
  statusInactiveOnWeak: undefined,
  statusInactiveStrong: undefined,
  statusInactiveWeak: undefined,
  statusPending: 'rgba(102, 102, 204, 1.0)',
  statusPendingOn: 'rgba(255, 255, 255, 1.0)',
  statusPendingOnStrong: undefined,
  statusPendingOnWeak: undefined,
  statusPendingStrong: undefined,
  statusPendingWeak: undefined,
  statusPotential: 'rgba(255, 255, 255, 0)',
  statusSuccess: 'rgba(0, 153, 0, 1.0)',
  statusSuccessOn: 'rgba(255, 255, 255, 1.0)',
  statusSuccessOnStrong: undefined,
  statusSuccessOnWeak: undefined,
  statusSuccessStrong: undefined,
  targetFocus: '0px 0px 4px rgba(0, 155, 222, 0.12)',
  statusSuccessWeak: undefined,
  strokeWidth: '1px',
  statusWarning: 'rgba(204, 153, 0, 1.0)',
  strokeWidthStrong: '3px',
  statusWarningOn: 'rgba(255, 255, 255, 1.0)',
  strokeWidthWeak: '0.5px',
  statusWarningOnStrong: undefined,
  statusWarningOnWeak: undefined,
  statusWarningStrong: undefined,
  statusWarningWeak: undefined,
};

const themeLight = {
  // Area and surface styles.
  background: 'rgba(240, 240, 240, 1.0)',
  backgroundBorder: undefined,
  backgroundOn: 'rgba(18, 18, 18, 1.0)',

  // Color schemes
  colorScheme: 'light',
  elevation: '0px 6px 12px rgba(0, 0, 0, 0.12)',
  elevationStrong: '0px 12px 24px rgba(0, 0, 0, 0.12)',
  elevationWeak: '0px 2px 4px rgba(0, 0, 0, 0.24)',
  float: 'rgba(255, 255, 255, 1.0)',
  floatBorder: 'rgba(225, 225, 225, 1.0)',
  floatOn: 'rgba(49, 49, 49, 1.0)',

  // Interactive styles.
  ix: 'rgba(255, 255, 255, 1.0)',
  ixActive: 'rgba(231, 247, 255, 1.0)',
  ixBorder: 'rgba(217, 217, 217, 1.0)',
  ixBorderActive: 'rgba(0, 155, 222, 1.0)',
  ixBorderInactive: 'rgba(217, 217, 217, 1.0)',
  ixCancel: 'rgba(89,89,89,1)',
  ixInactive: 'rgba(245, 245, 245, 1.0)',
  ixOn: 'rgba(38, 38, 38, 1.0)',
  ixOnActive: 'rgba(0, 155, 222, 1.0)',
  ixOnInactive: 'rgba(217, 217, 217, 1.0)',

  // Specialized and unique styles.
  overlay: 'rgba(255, 255, 255, 0.75)',
  overlayStrong: 'rgba(255, 255, 255, 1.0)',
  overlayWeak: 'rgba(255, 255, 255, 0.5)',
  stage: 'rgba(246, 246, 246, 1.0)',
  stageBorder: 'rgba(194, 194, 194, 1.0)',
  stageOn: 'rgba(69, 69, 69, 1.0)',

  // Palette colors for strong/weak calculations.
  strong: 'rgba(0, 0, 0, 1.0)',
  surface: 'rgba(250, 250, 250, 1.0)',
  surfaceBorder: 'rgba(212, 212, 212, 1.0)',
  surfaceOn: 'rgba(0, 8, 16, 1.0)',
  weak: 'rgba(255, 255, 255, 1.0)',
};

const themeDark = {
  // Area and surface styles.
  background: 'rgba(21, 21, 23, 1.0)',
  backgroundBorder: undefined,
  backgroundOn: 'rgba(237, 237, 237, 1.0)',

  // Color schemes
  colorScheme: 'dark',
  elevation: '0px 6px 12px rgba(255, 255, 255, 0.06)',
  elevationStrong: '0px 12px 24px rgba(255, 255, 255, 0.06)',
  elevationWeak: '0px 2px 4px rgba(255, 255, 255, 0.12)',
  float: 'rgba(60, 61, 62, 1.0)',
  floatBorder: 'rgba(90, 91, 92, 1.0)',
  floatOn: 'rgba(206, 206, 206, 1.0)',

  // Interactive styles.
  ix: 'rgba(21, 21, 23, 1.0)',
  ixActive: 'rgba(17, 27, 38, 1.0)',
  ixBorder: 'rgba(67, 67, 67, 1.0)',
  ixBorderActive: 'rgba(23, 125, 220, 1.0)',
  ixBorderInactive: 'rgba(80, 80, 80, 1.0)',
  ixCancel: 'rgba(115,115,115,1)',
  ixInactive: 'rgba(49, 49, 49, 1.0)',
  ixOn: 'rgba(209, 209, 209, 1.0)',
  ixOnActive: 'rgba(23, 125, 220, 1.0)',
  ixOnInactive: 'rgba(80, 80, 80, 1.0)',

  // Specialized and unique styles.
  overlay: 'rgba(0, 0, 0, 0.75)',
  overlayStrong: 'rgba(0, 0, 0, 1.0)',
  overlayWeak: 'rgba(0, 0, 0, 0.5)',
  stage: 'rgba(35, 36, 38, 1.0)',
  stageBorder: 'rgba(61, 61, 61, 1.0)',
  stageOn: 'rgba(186, 186, 186, 1.0)',

  // Palette colors for strong/weak calculations.
  strong: 'rgba(255, 255, 255, 1.0)',
  surface: 'rgba(48, 49, 50, 1.0)',
  surfaceBorder: 'rgba(85, 85, 85, 1.0)',
  surfaceOn: 'rgba(255, 247, 239, 1.0)',
  weak: 'rgba(0, 0, 0, 1.0)',
};

export const themeLightDetermined: Theme = generateStrongWeak(
  Object.assign({}, themeBase, themeLight),
);
export const themeDarkDetermined: Theme = generateStrongWeak(
  Object.assign({}, themeBase, themeDark),
);
const themeHpe = { brand: 'rgba(1, 169, 130, 1.0)' };

export const themeLightHpe: Theme = generateStrongWeak(
  Object.assign({}, themeBase, themeLight, themeHpe),
);
export const themeDarkHpe: Theme = generateStrongWeak(
  Object.assign({}, themeBase, themeDark, themeHpe),
);
