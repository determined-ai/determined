/* eslint-disable sort-keys-fix/sort-keys-fix */
import { RefObject } from 'react';

import { findParentByClass } from 'components/kit/internal/functions';
import { Theme } from 'components/kit/internal/theme';
import { ValueOf } from 'components/kit/internal/types';

export type { Theme };
export const MATCH_MEDIA_SCHEME_DARK = '(prefers-color-scheme: dark)';
export const MATCH_MEDIA_SCHEME_LIGHT = '(prefers-color-scheme: light)';

export const globalCssVars = {
  animationCurve: '0.2s cubic-bezier(0.785, 0.135, 0.15, 0.86)',

  iconBig: '28px',
  iconEnormous: '40px',
  iconGiant: '44px',
  iconGreat: '32px',
  iconHuge: '36px',
  iconJumbo: '48px',
  iconLarge: '24px',
  iconMedium: '20px',
  iconMega: '52px',
  iconSmall: '16px',
  iconTiny: '12px',

  navBottomBarHeight: '56px',
  navSideBarWidthMax: '240px',
  navSideBarWidthMin: '56px',
};

export const Mode = {
  System: 'system',
  Light: 'light',
  Dark: 'dark',
} as const;

export type Mode = ValueOf<typeof Mode>;

export const getCssVar = (ref: RefObject<HTMLElement>, name: string): string => {
  const varName = name.replace(/^(var\()?(.*?)\)?$/i, '$2');
  const element = ref.current || document.documentElement;
  return window
    .getComputedStyle(findParentByClass(element, 'ui-provider'))
    ?.getPropertyValue(varName);
};

export const getSystemMode = (): Mode => {
  const isDark = matchMedia?.(MATCH_MEDIA_SCHEME_DARK).matches;
  if (isDark) return Mode.Dark;

  const isLight = matchMedia?.(MATCH_MEDIA_SCHEME_LIGHT).matches;
  if (isLight) return Mode.Light;

  return Mode.System;
};
