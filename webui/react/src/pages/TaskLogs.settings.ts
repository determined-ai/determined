import { BaseType, SettingsConfig } from 'hooks/useSettings';

import { LogLevelFromApi } from './TaskLogFilters';

export interface Settings {
  agentId?: string[];
  allocationId?: string[];
  containerId?: string[];
  level?: LogLevelFromApi[];
  rankId?: number[];
}

const config: SettingsConfig = {
  settings: [
    {
      key: 'allocationId',
      storageKey: 'allocationId',
      type: {
        baseType: BaseType.String,
        isArray: true,
      },
    },
    {
      key: 'agentId',
      storageKey: 'agentId',
      type: {
        baseType: BaseType.String,
        isArray: true,
      },
    },
    {
      key: 'containerId',
      storageKey: 'containerId',
      type: {
        baseType: BaseType.String,
        isArray: true,
      },
    },
    {
      key: 'rankId',
      storageKey: 'rankId',
      type: {
        baseType: BaseType.Integer,
        isArray: true,
      },
    },
    {
      key: 'level',
      storageKey: 'level',
      type: {
        baseType: BaseType.String,
        isArray: true,
      },
    },
  ],
  storagePath: 'task-logs',
};

export default config;
