import { BaseType, SettingsConfig } from 'hooks/useSettings';

export interface Settings {
  theme: string;
}

export enum ThemeClass {
  SYSTEM = 'system',
  LIGHT = 'light',
  DARK = 'dark'
}

export const config: SettingsConfig = {
  settings: [
    {
      defaultValue: ThemeClass.SYSTEM,
      key: 'theme',
      storageKey: 'theme',
      type: { baseType: BaseType.String },
    },
  ],
  storagePath: 'settings/theme',
};
