import { Dispatch, SetStateAction, useCallback, useEffect, useMemo, useState } from 'react';

import { RecordKey } from 'shared/types';
import { camelCaseToKebab } from 'shared/utils/string';
import themes, {globalCssVars, Theme, DarkLight } from 'themes';
import { BrandingType } from 'types';
import { Mode } from 'components/ThemeToggle.settings';

type ThemeHook = {
  themeMode: DarkLight,
  mode: Mode,
  setBranding: Dispatch<SetStateAction<BrandingType>>,
  setMode: Dispatch<SetStateAction<Mode>>,
  theme: Theme,
};

const STYLESHEET_ID = 'antd-stylesheet';

const MATCH_MEDIA_SCHEME_DARK = '(prefers-color-scheme: dark)';
const MATCH_MEDIA_SCHEME_LIGHT = '(prefers-color-scheme: light)';

const themeConfig = {
  [Mode.DARK]: { antd: 'antd.dark.min.css' },
  [Mode.LIGHT]: { antd: 'antd.min.css' },
};

const getThemeType = (mode: Mode): DarkLight => mode === Mode.LIGHT ? DarkLight.Light : DarkLight.Dark; 

const createStylesheetLink = () => {
  const link = document.createElement('link');
  link.id = STYLESHEET_ID;
  link.rel = 'stylesheet';

  document.head.appendChild(link);

  return link;
};

const getStylesheetLink = () => {
  return document.getElementById(STYLESHEET_ID) as HTMLLinkElement || createStylesheetLink();
};

const getSystemMode = (): Mode => {
  const isDark = matchMedia?.(MATCH_MEDIA_SCHEME_DARK).matches;
  if (isDark) return Mode.DARK;

  const isLight = matchMedia?.(MATCH_MEDIA_SCHEME_LIGHT).matches;
  if (isLight) return Mode.LIGHT;

  return Mode.SYSTEM;
};

const updateAntDesignTheme = (path: string) => {
  const link = getStylesheetLink();
  link.href = `${process.env.PUBLIC_URL}/themes/${path}`;
};



/*
 * `useTheme` hook takes a `themeId` and converts the theme object and translates into
 * CSS variables that are applied throughout various component CSS modules. Upon a change
 * in the `themeId`, the hook dynamically updates the CSS variables once again.
 * `useTheme` hook is meant to be used only once in the top level component such as App
 * and storybook Theme decorators and not individual components.
 */
export const useTheme = (): ThemeHook => {
  const currentMode = getSystemMode();
  const [ branding, setBranding ] = useState(BrandingType.Determined);
  const [ mode, setMode ] = useState<Mode>(currentMode);
  const [ systemMode, setSystemMode ] = useState<Mode>(currentMode);
  const  themeMode = getThemeType(mode === Mode.SYSTEM ? systemMode === Mode.SYSTEM ? Mode.LIGHT : systemMode : mode);
  const theme = useMemo(() => themes[branding][themeMode], [ branding, mode ]);

  const handleSchemeChange = useCallback((event: MediaQueryListEvent) => {
    setSystemMode(event.matches ? Mode.DARK : Mode.LIGHT);
  }, []);

  useEffect(() => {
    // Set global CSS variables shared across themes.
    Object.keys(globalCssVars).forEach(key => {
      const value = (globalCssVars as Record<RecordKey, string>)[key];
      document.documentElement.style.setProperty(`--${camelCaseToKebab(key)}`, value);
    });

    // Set each theme property as top level CSS variable.
    Object.keys(theme).forEach(key => {
      const value = (theme as Record<RecordKey, string>)[key];
      document.documentElement.style.setProperty(`--theme-${camelCaseToKebab(key)}`, value);
    });
  }, [ theme ]);

  // Detect browser/OS level dark/light mode changes.
  useEffect(() => {
    matchMedia?.(MATCH_MEDIA_SCHEME_DARK).addEventListener('change', handleSchemeChange);

    return () => {
      matchMedia?.(MATCH_MEDIA_SCHEME_DARK).removeEventListener('change', handleSchemeChange);
    };
  }, [ handleSchemeChange ]);

  // When mode changes update theme.
  useEffect(() => {
    updateAntDesignTheme(themeConfig[themeMode].antd);
  }, [ themeMode ]);

  return { mode, themeMode, setBranding, setMode, theme };
};

export default useTheme;
