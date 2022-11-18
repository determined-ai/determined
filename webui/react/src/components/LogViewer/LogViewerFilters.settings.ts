import { BaseType, SettingsConfig } from 'hooks/useSettings';
import { LogLevelFromApi } from 'types';

export interface Settings {
  agentId?: string[];
  allocationId?: string[];
  containerId?: string[];
  level?: LogLevelFromApi[];
  rankId?: number[];
  searchText?: string;
}

const config: SettingsConfig = {
  settings: [
    {
      key: 'allocationId',
      type: {
        baseType: BaseType.String,
        isArray: true,
      },
    },
    {
      key: 'agentId',
      type: {
        baseType: BaseType.String,
        isArray: true,
      },
    },
    {
      key: 'containerId',
      type: {
        baseType: BaseType.String,
        isArray: true,
      },
    },
    {
      key: 'rankId',
      type: {
        baseType: BaseType.Integer,
        isArray: true,
      },
    },
    {
      key: 'level',
      type: {
        baseType: BaseType.String,
        isArray: true,
      },
    },
    {
      key: 'searchText',
      type: { baseType: BaseType.String },
    },
  ],
  storagePath: 'log-viewer-filters',
};

export default config;
