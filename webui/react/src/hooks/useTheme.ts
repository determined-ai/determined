import { Dispatch, SetStateAction, useEffect, useState } from 'react';

import themes, { defaultThemeId, ThemeId } from 'themes';
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

/*
 * `useTheme` hook takes a `themeId` and converts the theme object and translates into
 * CSS variables that are applied throughout various component CSS modules. Upon a change
 * in the `themeId`, the hook dynamically updates the CSS variables once again.
 * `useTheme` hook is meant to be used only once in the top level component such as App
 * and storybook Theme decorators and not individual components.
 */
export const useTheme = (): { dispatch: Dispatch<SetStateAction<ThemeId>>; state: ThemeId } => {
  const [ themeId, setThemeId ] = useState<ThemeId>(defaultThemeId);

  useEffect(() => {
    const theme = themes[themeId];
    const root = document.documentElement;
    const cssVars = flattenTheme(theme);

    // Set each theme property as top level CSS variable
    Object.keys(cssVars).forEach(key => root.style.setProperty(`--${key}`, cssVars[key]));
  }, [ themeId ]);

  return { dispatch: setThemeId, state: themeId };
};

export default useTheme;
