import { SettingsConfig } from 'hooks/useSettings';
import { KeyboardShortcut } from 'utils/shortcut';

export interface Settings {
  navbarCollapsed: KeyboardShortcut;
}

const shortCutSettingsConfig: SettingsConfig<Settings> = {
  settings: {
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
