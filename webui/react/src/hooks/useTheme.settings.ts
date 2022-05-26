import { BaseType, SettingsConfig } from 'hooks/useSettings';

export enum Mode {
  SYSTEM = 'system',
  LIGHT = 'light',
  DARK = 'dark'
}

export interface Settings {
  theme: Mode;
}


export const config: SettingsConfig = {
  settings: [
    {
      defaultValue: Mode.SYSTEM,
      key: 'theme',
      storageKey: 'theme',
      type: { baseType: BaseType.String },
    },
  ],
  storagePath: 'settings/theme',
};
