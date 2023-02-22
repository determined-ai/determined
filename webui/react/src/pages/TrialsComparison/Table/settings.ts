import { array, boolean, number, string, undefined, union } from 'io-ts';

import { InteractiveTableSettings } from 'components/Table/InteractiveTable';
import { SettingsConfig } from 'hooks/useSettings';

export const trialsTableSettingsConfig: SettingsConfig<InteractiveTableSettings> = {
  settings: {
    columns: {
      defaultValue: [],
      skipUrlEncoding: true,
      storageKey: 'columns',
      type: array(string),
    },
    columnWidths: {
      defaultValue: [],
      skipUrlEncoding: true,
      storageKey: 'columnWidths',
      type: array(number),
    },
    sortDesc: {
      defaultValue: true,
      skipUrlEncoding: true,
      storageKey: 'sortDesc',
      type: boolean,
    },
    sortKey: {
      defaultValue: 'trialId',
      skipUrlEncoding: true,
      storageKey: 'sortKey',
      type: union([undefined, string, number, boolean]),
    },
    tableLimit: {
      defaultValue: 20,
      skipUrlEncoding: true,
      storageKey: 'tableLimit',
      type: number,
    },
    tableOffset: {
      defaultValue: 0,
      skipUrlEncoding: true,
      storageKey: 'tableOffset',
      type: number,
    },
  },
  storagePath: 'trial-table',
};
