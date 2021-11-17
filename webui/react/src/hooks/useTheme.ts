import { Dispatch, SetStateAction, useEffect, useMemo, useState } from 'react';

import themes, { DarkLight, Theme } from 'themes';
import { BrandingType } from 'types';
import { getIsDarkMode } from 'utils/browser';
import { isObject } from 'utils/data';

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
type SubTheme = Record<string, any>;

const flattenTheme = (theme: SubTheme, basePath = 'theme'): SubTheme => {
  if (isObject(theme)) {
    return Object.keys(theme)
      .map(key => flattenTheme(theme[key], `${basePath}-${key}`))
      .reduce((acc, sub) => ({ ...acc, ...sub }), {});
  } else if (Array.isArray(theme)) {
    return theme
      .map((sub: SubTheme, index: number) => flattenTheme(sub, `${basePath}-${index}`))
      .reduce((acc, sub) => ({ ...acc, ...sub }), {});
  }
  return { [basePath]: theme };
};

type ThemeHook = {
  setBranding: Dispatch<SetStateAction<BrandingType>>,
  setMode: Dispatch<SetStateAction<DarkLight>>,
  theme: Theme,
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

  useEffect(() => {
    const root = document.documentElement;
    const cssVars = flattenTheme(theme);

    // Set each theme property as top level CSS variable
    Object.keys(cssVars).forEach(key => root.style.setProperty(`--${key}`, cssVars[key]));
  }, [ theme ]);

  return { setBranding, setMode, theme };
};

export default useTheme;
