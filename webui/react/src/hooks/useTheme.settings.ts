import { BaseType, SettingsConfig } from 'hooks/useSettings';

export enum Mode {
  System = 'system',
  Light = 'light',
  Dark = 'dark'
}

export interface Settings {
  theme: Mode;
}

export const config: SettingsConfig = {
  settings: [
    {
      defaultValue: Mode.System,
      key: 'theme',
      storageKey: 'theme',
      type: { baseType: BaseType.String },
    },
  ],
  storagePath: 'settings/theme',
};
