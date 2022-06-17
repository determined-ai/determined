import { useCallback, useEffect, useState } from 'react';

import { StoreAction, useStore, useStoreDispatch } from 'contexts/Store';
import useSettings from 'hooks/useSettings';
import { DarkLight, globalCssVars, Theme } from 'shared/themes';
import { RecordKey } from 'shared/types';
import { camelCaseToKebab } from 'shared/utils/string';
import themes from 'themes';
import { BrandingType, DarkLight, Mode } from 'types';

import { config, Settings } from './useTheme.settings';

type ThemeHook = {
  // mode: Mode,
  // setMode: Dispatch<SetStateAction<Mode>>,
  // theme: Theme,
  // themeMode: DarkLight,
  // themeSetting: string,
  // updateTheme: (mode: Mode) => void
};

const STYLESHEET_ID = 'antd-stylesheet';
const MATCH_MEDIA_SCHEME_DARK = '(prefers-color-scheme: dark)';
const MATCH_MEDIA_SCHEME_LIGHT = '(prefers-color-scheme: light)';
const ANTD_THEMES = {
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

const getDarkLight = (mode: Mode, systemMode: Mode): DarkLight => {
  const resolvedMode = mode === Mode.System
    ? (systemMode === Mode.System ? Mode.Light : systemMode)
    : mode;
  return resolvedMode === Mode.Light ? DarkLight.Light : DarkLight.Dark;
};

const getStylesheetLink = () => {
  return document.getElementById(STYLESHEET_ID) as HTMLLinkElement || createStylesheetLink();
};

const getSystemMode = (): Mode => {
  const isDark = matchMedia?.(MATCH_MEDIA_SCHEME_DARK).matches;
  if (isDark) return Mode.Dark;

  const isLight = matchMedia?.(MATCH_MEDIA_SCHEME_LIGHT).matches;
  if (isLight) return Mode.Light;

  return Mode.System;
};

const updateAntDesignTheme = (path: string) => {
  const link = getStylesheetLink();
  link.href = `${process.env.PUBLIC_URL}/themes/${path}`;
};

/**
 * `useTheme` hook takes a `themeId` and converts the theme object and translates into
 * CSS variables that are applied throughout various component CSS modules. Upon a change
 * in the `themeId`, the hook dynamically updates the CSS variables once again.
 * `useTheme` hook is meant to be used only once in the top level component such as App
 * and storybook Theme decorators and not individual components.
*/
export const useTheme = (): void => {
  const [ systemMode, setSystemMode ] = useState<Mode>(getSystemMode());
  const { info, ui } = useStore();
  const storeDispatch = useStoreDispatch();
  const {
    settings,
    updateSettings,
  } = useSettings<Settings>(config);

  const handleSchemeChange = useCallback((event: MediaQueryListEvent) => {
    if (!event.matches) setSystemMode(getSystemMode());
  }, []);

  // const themeSetting = settings.theme;
  // const updateTheme = (mode: Mode) => {
  //   setMode(mode);
  //   updateSettings({ theme: mode });
  // };

  useEffect(() => {
    // Set global CSS variables shared across themes.
    Object.keys(globalCssVars).forEach(key => {
      const value = (globalCssVars as Record<RecordKey, string>)[key];
      document.documentElement.style.setProperty(`--${camelCaseToKebab(key)}`, value);
    });

    // Set each theme property as top level CSS variable.
    Object.keys(ui.theme).forEach(key => {
      const value = (ui.theme as Record<RecordKey, string>)[key];
      document.documentElement.style.setProperty(`--theme-${camelCaseToKebab(key)}`, value);
    });
  }, [ ui.theme ]);

  // Detect browser/OS level dark/light mode changes.
  useEffect(() => {
    matchMedia?.(MATCH_MEDIA_SCHEME_DARK).addEventListener('change', handleSchemeChange);
    matchMedia?.(MATCH_MEDIA_SCHEME_LIGHT).addEventListener('change', handleSchemeChange);

    return () => {
      matchMedia?.(MATCH_MEDIA_SCHEME_DARK).removeEventListener('change', handleSchemeChange);
      matchMedia?.(MATCH_MEDIA_SCHEME_LIGHT).addEventListener('change', handleSchemeChange);
    };
  }, [ handleSchemeChange ]);

  // Update Ant Design theme when darkLight changes.
  useEffect(() => updateAntDesignTheme(ANTD_THEMES[ui.darkLight].antd), [ ui.darkLight ]);

  // Update darkLight and theme when branding, system mode, or mode changes .
  useEffect(() => {
    const darkLight = getDarkLight(ui.mode, systemMode);
    storeDispatch({
      type: StoreAction.SetTheme,
      value: { darkLight, theme: themes[info.branding][darkLight] },
    });
  }, [ info.branding, storeDispatch, systemMode, ui.mode ]);
};

export default useTheme;
