import { MINIMUM_PAGE_SIZE } from 'components/Table';
import { BaseType, SettingsConfig } from 'hooks/useSettings';
import { V1GetExperimentTrialsRequestSortBy } from 'services/api-ts-sdk';
import { RunState } from 'types';

export interface Settings {
  compare: boolean;
  row?: number[];
  sortDesc: boolean;
  sortKey: V1GetExperimentTrialsRequestSortBy;
  state?: RunState[];
  tableLimit: number;
  tableOffset: number;
}

const config: SettingsConfig = {
  settings: [
    {
      defaultValue: false,
      key: 'compare',
      type: { baseType: BaseType.Boolean },
    },
    {
      key: 'row',
      type: { baseType: BaseType.Integer, isArray: true },
    },
    {
      defaultValue: true,
      key: 'sortDesc',
      storageKey: 'sortDesc',
      type: { baseType: BaseType.Boolean },
    },
    {
      defaultValue: V1GetExperimentTrialsRequestSortBy.ID,
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
  ],
  storagePath: 'experiment-trials-list',
};

export default config;
