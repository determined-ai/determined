import { literal, union } from 'io-ts';

import { Mode } from 'components/ThemeProvider';
import { SettingsConfig } from 'hooks/useSettings';

export interface Settings {
  mode: Mode;
}

export const config: SettingsConfig<Settings> = {
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
