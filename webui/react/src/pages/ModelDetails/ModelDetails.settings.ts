import { MINIMUM_PAGE_SIZE } from 'components/Table';
import { BaseType, SettingsConfig } from 'hooks/useSettings';
import { V1GetModelVersionsRequestSortBy } from 'services/api-ts-sdk';

export interface Settings {
  sortDesc: boolean;
  sortKey: V1GetModelVersionsRequestSortBy;
  tableLimit: number;
  tableOffset: number;
}

const config: SettingsConfig = {
  settings: [
    {
      defaultValue: true,
      key: 'sortDesc',
      storageKey: 'sortDesc',
      type: { baseType: BaseType.Boolean },
    },
    {
      defaultValue: V1GetModelVersionsRequestSortBy.VERSION,
      key: 'sortKey',
      storageKey: 'sortKey',
      type: { baseType: BaseType.String },
    },
    {
      defaultValue: MINIMUM_PAGE_SIZE,
      key: 'tableLimit',
      storageKey: 'tableLimit',
      type: { baseType: BaseType.Integer },
    },
    {
      defaultValue: 0,
      key: 'tableOffset',
      type: { baseType: BaseType.Integer },
    },
  ],
  storagePath: 'model-details',
};

export default config;
