import { literal, union } from 'io-ts';

import { SettingsConfig } from 'hooks/useSettings';
import { Mode } from 'shared/themes';

export interface Settings {
  mode: Mode;
}

export const config: SettingsConfig<Settings> = {
  applicableRoutespace: 'settings/theme',
  settings: {
    mode: {
      defaultValue: Mode.System,
      skipUrlEncoding: true,
      storageKey: 'mode',
      type: union([literal(Mode.Dark), literal(Mode.Light), literal(Mode.System)]),
    },
  },
  storagePath: 'settings-theme',
};
