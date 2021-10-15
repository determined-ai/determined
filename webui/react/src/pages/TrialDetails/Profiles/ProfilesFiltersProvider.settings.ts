import { BaseType, SettingsConfig } from 'hooks/useSettings';

export interface Settings {
  agentId?: string;
  gpuUuid?: string;
  name?: string;
}

const config: SettingsConfig = {
  settings: [
    {
      key: 'name',
      type: { baseType: BaseType.String },
    },
    {
      key: 'agentId',
      type: { baseType: BaseType.String },
    },
    {
      key: 'gpuUuid',
      type: { baseType: BaseType.String },
    },
  ],
  storagePath: 'profiler-filters',
};

export default config;
