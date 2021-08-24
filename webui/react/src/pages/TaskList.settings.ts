import { MINIMUM_PAGE_SIZE } from 'components/Table';
import { BaseType, SettingsConfig } from 'hooks/useSettings';
import { CommandState, CommandType } from 'types';

export interface Settings {
  row?: string[];
  search?: string;
  sortDesc: boolean;
  sortKey: 'id' | 'name' | 'resourcePool' | 'startTime' | 'state' | 'type' | 'username';
  state?: CommandState[];
  tableLimit: number;
  tableOffset: number;
  type?: CommandType[];
  user?: string[];
}

const config: SettingsConfig = {
  settings: [
    {
      key: 'row',
      type: { baseType: BaseType.String, isArray: true },
    },
    {
      key: 'search',
      type: { baseType: BaseType.String },
    },
    {
      defaultValue: true,
      key: 'sortDesc',
      storageKey: 'sortDesc',
      type: { baseType: BaseType.Boolean },
    },
    {
      defaultValue: 'startTime',
      key: 'sortKey',
      storageKey: 'sortKey',
      type: { baseType: BaseType.String },
    },
    {
      key: 'state',
      storageKey: 'state',
      type: {
        baseType: BaseType.String,
        isArray: true,
      },
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
      key: 'type',
      storageKey: 'type',
      type: {
        baseType: BaseType.String,
        isArray: true,
      },
    },
    {
      key: 'user',
      storageKey: 'user',
      type: {
        baseType: BaseType.String,
        isArray: true,
      },
    },
  ],
  storagePath: 'task-list',
};

export default config;
