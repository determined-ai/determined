import { BaseType, SettingsConfig } from 'hooks/useSettings';

export const trialsTableSettingsConfig: SettingsConfig = {
  applicableRoutespace: '/trials',
  settings: [
    {
      defaultValue: [],
      key: 'columns',
      skipUrlEncoding: true,
      storageKey: 'columns',
      type: {
        baseType: BaseType.String,
        isArray: true,
      },
    },
    {
      defaultValue: [],
      key: 'columnWidths',
      skipUrlEncoding: true,
      storageKey: 'columnWidths',
      type: {
        baseType: BaseType.Float,
        isArray: true,
      },
    },
    {
      defaultValue: true,
      key: 'sortDesc',
      skipUrlEncoding: true,
      storageKey: 'sortDesc',
      type: { baseType: BaseType.Boolean },
    },
    {
      defaultValue: 'trialId',
      key: 'sortKey',
      skipUrlEncoding: true,
      storageKey: 'sortKey',
      type: { baseType: BaseType.String },
    },
    {
      defaultValue: 20,
      key: 'tableLimit',
      skipUrlEncoding: true,
      storageKey: 'tableLimit',
      type: { baseType: BaseType.Integer },
    },
    {
      defaultValue: 0,
      key: 'tableOffset',
      skipUrlEncoding: true,
      storageKey: 'tableOffset',
      type: { baseType: BaseType.Integer },
    },
  ],
  storagePath: 'trial-table',
};
