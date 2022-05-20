import { Dispatch, SetStateAction, useCallback, useEffect, useMemo, useState } from 'react';

import { RecordKey } from 'shared/types';
import { camelCaseToKebab } from 'shared/utils/string';
import themes, { DarkLight, globalCssVars, Theme } from 'themes';
import { BrandingType } from 'types';

type ThemeHook = {
  mode: DarkLight,
  setBranding: Dispatch<SetStateAction<BrandingType>>,
  setMode: Dispatch<SetStateAction<DarkLight>>,
  theme: Theme,
};

const STYLESHEET_ID = 'antd-stylesheet';
const MATCH_MEDIA_SCHEME_DARK = '(prefers-color-scheme: dark)';

const themeConfig = {
  [DarkLight.Dark]: { antd: 'antd.dark.min.css' },
  [DarkLight.Light]: { antd: 'antd.min.css' },
};

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

const getIsDarkMode = (): boolean => {
  return matchMedia?.(MATCH_MEDIA_SCHEME_DARK).matches;
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
  const [ branding, setBranding ] = useState(BrandingType.Determined);
  const [ mode, setMode ] = useState(() => getIsDarkMode() ? DarkLight.Dark : DarkLight.Light);

  const theme = useMemo(() => themes[branding][mode], [ branding, mode ]);

  const handleSchemeChange = useCallback((event: MediaQueryListEvent) => {
    setMode(event.matches ? DarkLight.Dark : DarkLight.Light);
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
    updateAntDesignTheme(themeConfig[mode].antd);
  }, [ mode ]);

  return { mode, setBranding, setMode, theme };
};

export default useTheme;
