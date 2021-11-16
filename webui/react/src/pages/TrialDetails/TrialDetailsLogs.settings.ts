import { BaseType, SettingsConfig } from 'hooks/useSettings';

import { LogLevelFromApi } from './Logs/TrialLogFilters';

export interface Settings {
  agentId?: string[];
  containerId?: string[];
  level?: LogLevelFromApi[];
  rankId?: number[];
}

const config: SettingsConfig = {
  settings: [
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
  storagePath: 'trial-logs',
};

export default config;
