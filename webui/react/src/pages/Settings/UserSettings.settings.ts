import { SettingsConfig } from 'hooks/useSettings';
import { KeyboardShortcut } from 'utils/shortcut';

export interface Settings {
  navbarCollapsed: KeyboardShortcut;
  jupyterLab: KeyboardShortcut;
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
  },
  storagePath: 'shortcuts',
};

export default shortCutSettingsConfig;
