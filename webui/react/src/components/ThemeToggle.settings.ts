import { BaseType, SettingsConfig } from 'hooks/useSettings';

export interface Settings {
  theme: string;
}

export enum Mode {
  SYSTEM = 'system',
  LIGHT = 'light',
  DARK = 'dark'
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
