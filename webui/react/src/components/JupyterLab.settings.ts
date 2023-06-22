import { number, string, undefined as undefinedType, union } from 'io-ts';

import { SettingsConfig } from 'hooks/useSettings';
import { JupyterLabOptions } from 'utils/jupyter';
import { KeyboardShortcut } from 'utils/shortcut';

const STORAGE_PATH = 'jupyter-lab';
export const DEFAULT_SLOT_COUNT = 1;

const defaultShortcut: KeyboardShortcut = {
  alt: false,
  ctrl: false,
  key: 'L',
  meta: true,
  shift: true,
};

const JupyterLabSettings: SettingsConfig<JupyterLabOptions> = {
  settings: {
    name: {
      defaultValue: '',
      skipUrlEncoding: true,
      storageKey: 'name',
      type: union([string, undefinedType]),
    },
    pool: {
      defaultValue: '',
      skipUrlEncoding: true,
      storageKey: 'pool',
      type: union([string, undefinedType]),
    },
    shortcut: {
      defaultValue: JSON.stringify(defaultShortcut),
      skipUrlEncoding: true,
      storageKey: 'shortcut',
      type: union([string, undefinedType]),
    },
    slots: {
      defaultValue: DEFAULT_SLOT_COUNT,
      skipUrlEncoding: true,
      storageKey: 'slots',
      type: union([number, undefinedType]),
    },
    template: {
      defaultValue: undefined,
      skipUrlEncoding: true,
      storageKey: 'template',
      type: union([string, undefinedType]),
    },
    workspaceId: {
      defaultValue: undefined,
      skipUrlEncoding: true,
      storageKey: 'workspaceId',
      type: union([number, undefinedType]),
    },
  },
  storagePath: STORAGE_PATH,
};

export default JupyterLabSettings;
