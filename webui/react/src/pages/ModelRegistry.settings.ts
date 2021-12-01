import { MINIMUM_PAGE_SIZE } from 'components/Table';
import { BaseType, SettingsConfig } from 'hooks/useSettings';
import { V1GetModelsRequestSortBy } from 'services/api-ts-sdk';

export interface Settings {
  archived?: boolean;
  description?: string;
  name?: string;
  sortDesc: boolean;
  sortKey: V1GetModelsRequestSortBy;
  tableLimit: number;
  tableOffset: number;
  tags?: string[];
  users?: string[];
}

const config: SettingsConfig = {
  settings: [
    {
      key: 'archived',
      storageKey: 'archived',
      type: { baseType: BaseType.Boolean },
    },
    {
      defaultValue: true,
      key: 'sortDesc',
      storageKey: 'sortDesc',
      type: { baseType: BaseType.Boolean },
    },
    {
      defaultValue: V1GetModelsRequestSortBy.CREATIONTIME,
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
    {
      key: 'users',
      storageKey: 'users',
      type: {
        baseType: BaseType.String,
        isArray: true,
      },
    },
    {
      key: 'tags',
      storageKey: 'tags',
      type: {
        baseType: BaseType.String,
        isArray: true,
      },
    },
    {
      key: 'name',
      storageKey: 'name',
      type: { baseType: BaseType.String },
    },
    {
      key: 'description',
      storageKey: 'description',
      type: { baseType: BaseType.String },
    },
  ],
  storagePath: 'model-registry',
};

export default config;
