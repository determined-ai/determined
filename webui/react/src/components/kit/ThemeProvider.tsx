import { ConfigProvider, theme } from 'antd';
import { ThemeConfig } from 'antd/es/config-provider/context';
import React, { ReactNode, useCallback, useEffect, useLayoutEffect, useState } from 'react';

import useUI from 'components/kit/contexts/UI';
import themes from 'components/kit/themes';
import { DarkLight, globalCssVars, Mode } from 'components/kit/utils/themes';

import { BrandingType, RecordKey } from './internal/types';

const MATCH_MEDIA_SCHEME_DARK = '(prefers-color-scheme: dark)';
const MATCH_MEDIA_SCHEME_LIGHT = '(prefers-color-scheme: light)';
const ANTD_THEMES: Record<DarkLight, ThemeConfig> = {
  [DarkLight.Dark]: {
    algorithm: theme.darkAlgorithm,
    components: {
      Button: {
        colorBgContainer: 'transparent',
      },
      Checkbox: {
        colorBgContainer: 'transparent',
      },
      DatePicker: {
        colorBgContainer: 'transparent',
      },
      Input: {
        colorBgContainer: 'transparent',
      },
      InputNumber: {
        colorBgContainer: 'transparent',
      },
      Modal: {
        colorBgElevated: 'var(--theme-stage)',
      },
      Pagination: {
        colorBgContainer: 'transparent',
      },
      Progress: {
        marginXS: 0,
      },
      Radio: {
        colorBgContainer: 'transparent',
      },
      Select: {
        colorBgContainer: 'transparent',
      },
      Tree: {
        colorBgContainer: 'transparent',
      },
    },
    token: {
      borderRadius: 2,
      colorLink: '#57a3fa',
      colorLinkHover: '#8dc0fb',
      colorPrimary: '#1890ff',
      fontFamily: 'var(--theme-font-family)',
    },
  },
  [DarkLight.Light]: {
    algorithm: theme.defaultAlgorithm,
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
  },
};

const getDarkLight = (mode: Mode, systemMode: Mode): DarkLight => {
  const resolvedMode =
    mode === Mode.System ? (systemMode === Mode.System ? Mode.Light : systemMode) : mode;
  return resolvedMode === Mode.Light ? DarkLight.Light : DarkLight.Dark;
};

const getSystemMode = (): Mode => {
  const isDark = matchMedia?.(MATCH_MEDIA_SCHEME_DARK).matches;
  if (isDark) return Mode.Dark;

  const isLight = matchMedia?.(MATCH_MEDIA_SCHEME_LIGHT).matches;
  if (isLight) return Mode.Light;

  return Mode.System;
};

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
 * Wraps various theme settings together
 */
export const ThemeProvider: React.FC<{ children: ReactNode; branding?: BrandingType }> = ({
  children,
  branding,
}) => {
  const { ui, actions: uiActions } = useUI();
  const [systemMode, setSystemMode] = useState<Mode>(() => getSystemMode());

  const handleSchemeChange = useCallback((event: MediaQueryListEvent) => {
    if (!event.matches) setSystemMode(getSystemMode());
  }, []);

  useLayoutEffect(() => {
    // Set global CSS variables shared across themes.
    Object.keys(globalCssVars).forEach((key) => {
      const value = (globalCssVars as Record<RecordKey, string>)[key];
      document.documentElement.style.setProperty(`--${camelCaseToKebab(key)}`, value);
    });

    // Set each theme property as top level CSS variable.
    Object.keys(ui.theme).forEach((key) => {
      const value = (ui.theme as Record<RecordKey, string>)[key];
      document.documentElement.style.setProperty(`--theme-${camelCaseToKebab(key)}`, value);
    });
  }, [ui.theme]);

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
    const darkLight = getDarkLight(ui.mode, systemMode);
    uiActions.setTheme(darkLight, themes[branding || 'determined'][darkLight]);
  }, [branding, uiActions, systemMode, ui.mode]);

  const antdTheme = ANTD_THEMES[ui.darkLight];

  return <ConfigProvider theme={antdTheme}>{children}</ConfigProvider>;
};

export default ThemeProvider;
