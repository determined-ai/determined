import { BaseType, SettingsConfig } from 'hooks/useSettings';
import { Mode } from 'types';

export interface Settings {
  mode: Mode;
}

export const config: SettingsConfig = {
  settings: [
    {
      defaultValue: Mode.System,
      key: 'mode',
      skipUrlEncoding: true,
      storageKey: 'mode',
      type: { baseType: BaseType.String },
    },
  ],
  storagePath: 'settings/theme',
};
