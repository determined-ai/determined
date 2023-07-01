import { SettingsConfig } from 'hooks/useSettings';
import { KeyboardShortcut } from 'utils/shortcut';

export interface Settings {
  navbarCollapsed: KeyboardShortcut;
  jupyterLab: KeyboardShortcut;
  omnibar: KeyboardShortcut;
}

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
        key: 'u',
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
        key: ' ',
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
