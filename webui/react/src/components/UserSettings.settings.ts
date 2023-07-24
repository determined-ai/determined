import { type } from 'io-ts';

import { SettingsConfig } from 'hooks/useSettings';
import { KeyboardShortcut } from 'utils/shortcut';

export interface Settings {
  navbarCollapsed: KeyboardShortcut;
  jupyterLab: KeyboardShortcut;
  omnibar: KeyboardShortcut;
}

export const shortcutSettingsConfig = type({
  jupyterLab: KeyboardShortcut,
  navbarCollapsed: KeyboardShortcut,
  omnibar: KeyboardShortcut,
});

export const shortcutSettingsDefaults = {
  jupyterLab: {
    alt: false,
    ctrl: false,
    key: 'L',
    meta: true,
    shift: true,
  },
  navbarCollapsed: {
    alt: false,
    ctrl: false,
    key: 'U',
    meta: true,
    shift: true,
  },
  omnibar: {
    alt: false,
    ctrl: true,
    key: 'Space',
    meta: false,
    shift: false,
  },
} as const;

export const shortcutsSettingsPath = 'shortcuts';

const shortCutSettingsConfig: SettingsConfig<Settings> = {
  settings: {
    jupyterLab: {
      defaultValue: {
        alt: false,
        ctrl: false,
        key: 'L',
        meta: true,
        shift: true,
      },
      skipUrlEncoding: true,
      storageKey: 'jupyterLab',
      type: KeyboardShortcut,
    },
    navbarCollapsed: {
      defaultValue: {
        alt: false,
        ctrl: false,
        key: 'U',
        meta: true,
        shift: true,
      },
      skipUrlEncoding: true,
      storageKey: 'navbarCollapsed',
      type: KeyboardShortcut,
    },
    omnibar: {
      defaultValue: {
        alt: false,
        ctrl: true,
        key: 'Space',
        meta: false,
        shift: false,
      },
      skipUrlEncoding: true,
      storageKey: 'omnibar',
      type: KeyboardShortcut,
    },
  },
  storagePath: 'shortcuts',
};

export default shortCutSettingsConfig;
