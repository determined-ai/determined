import _ from 'lodash';
import { useObservable } from 'micro-observables';
import { useCallback, useEffect, useMemo, useState } from 'react';

import {
  themeDarkDetermined,
  themeDarkHpe,
  themeLightDetermined,
  themeLightHpe,
} from 'components/kit/internal/theme';
import { DarkLight } from 'components/kit/internal/types';
import {
  getSystemMode,
  MATCH_MEDIA_SCHEME_DARK,
  MATCH_MEDIA_SCHEME_LIGHT,
  Mode,
  Theme,
} from 'components/kit/Theme/themeUtils';
import determinedInfo, { BrandingType } from 'stores/determinedInfo';

const themes = {
  [BrandingType.Determined]: {
    [DarkLight.Dark]: themeDarkDetermined,
    [DarkLight.Light]: themeLightDetermined,
  },
  [BrandingType.HPE]: {
    [DarkLight.Dark]: themeDarkHpe,
    [DarkLight.Light]: themeLightHpe,
  },
};

const getDarkLight = (mode: Mode, systemMode: Mode): DarkLight => {
  const resolvedMode =
    mode === Mode.System ? (systemMode === Mode.System ? Mode.Light : systemMode) : mode;
  return resolvedMode === Mode.Light ? DarkLight.Light : DarkLight.Dark;
};

export const useTheme = (
  mode: Mode,
  userTheme: Theme,
): {
  theme: Theme;
  isDarkMode: boolean;
} => {
  const info = useObservable(determinedInfo.info);

  const branding = info?.branding || BrandingType.Determined;
  const [systemMode, setSystemMode] = useState<Mode>(() => getSystemMode());

  const darkLight = getDarkLight(mode, systemMode);
  const defaultTheme = useMemo(() => themes[branding][darkLight], [branding, darkLight]);

  const [theme, setTheme] = useState<Theme>(defaultTheme);
  const [isDarkMode, setIsDarkMode] = useState<boolean>(darkLight === DarkLight.Dark);

  useEffect(() => {
    setTheme(userTheme && !_.isEqual(userTheme, {}) ? userTheme : themes[branding][darkLight]);
  }, [userTheme, branding, darkLight]);

  useEffect(() => {
    setIsDarkMode(darkLight === DarkLight.Dark);
  }, [darkLight]);

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

  return { isDarkMode, theme };
};
