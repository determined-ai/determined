import { BaseType, SettingsConfig } from 'hooks/useSettings';
import { Mode } from 'types';

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
