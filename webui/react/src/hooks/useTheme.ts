import { literal, union } from 'io-ts';
import { useObservable } from 'micro-observables';
import { useLayoutEffect, useState } from 'react';

import useUI, { Mode, Theme } from 'components/kit/Theme';
import { DarkLight } from 'components/kit/Theme';
import { themes } from 'components/kit/Theme';
import { SettingsConfig } from 'hooks/useSettings';
import determinedStore, { BrandingType } from 'stores/determinedInfo';

export interface Settings {
  mode: Mode;
}

const MATCH_MEDIA_SCHEME_DARK = '(prefers-color-scheme: dark)';
const MATCH_MEDIA_SCHEME_LIGHT = '(prefers-color-scheme: light)';

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

export const config: SettingsConfig<Settings> = {
  settings: {
    mode: {
      defaultValue: Mode.System,
      skipUrlEncoding: true,
      storageKey: 'mode',
      type: union([literal(Mode.Dark), literal(Mode.Light), literal(Mode.System)]),
    },
  },
  storagePath: 'settings-theme',
};

const getTheme = (darkLight: DarkLight, branding?: BrandingType) =>
  themes[branding || 'determined'][darkLight];

const useTheme = (): { theme: Theme; isDarkMode: boolean } => {
  const info = useObservable(determinedStore.info);
  const {
    ui: { mode },
  } = useUI();

  const branding = info.branding;
  const systemMode = getSystemMode();
  const darkLight = getDarkLight(mode, systemMode);
  const [theme, setTheme] = useState<Theme>(getTheme(darkLight, branding));

  useLayoutEffect(() => {
    setTheme(getTheme(darkLight, branding));
  }, [mode, darkLight]);

  const isDarkMode = mode === DarkLight.Dark;

  return { isDarkMode, theme };
};

export default useTheme;
