import { Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useMemo } from 'react';

import SelectFilter from 'components/SelectFilter';
const { Option } = Select;
import { UpdateSettings } from 'hooks/useSettings';

import { AvailableSeriesType } from '../types';

import { Settings } from './SystemMetricChart';

export interface FiltersInterface {
  agentId?: string;
  gpuUuid?: string;
  name?: string;
}

interface Props {
  settings: Settings;
  systemSeries: AvailableSeriesType;
  updateSettings?: UpdateSettings<Settings>;
}

const SystemMetricFilter: React.FC<Props> = ({ settings, systemSeries, updateSettings }) => {
  const handleChangeAgentId = useCallback(
    (newAgentId: SelectValue) => {
      updateSettings?.({ agentId: newAgentId as unknown as string });
    },
    [updateSettings],
  );

  const handleChangeGpuUuid = useCallback(
    (newGpuUuid: SelectValue) => {
      updateSettings?.({ gpuUuid: newGpuUuid as unknown as string });
    },
    [updateSettings],
  );

  const handleChangeName = useCallback(
    (newName: SelectValue) => {
      updateSettings?.({ name: newName as unknown as string });
    },
    [updateSettings],
  );

  const uuidOptions = useMemo(() => {
    if (!settings.name || !settings.agentId) return [];
    return systemSeries?.[settings.name]?.[settings.agentId]?.filter((uuid) => !!uuid) || [];
  }, [settings, systemSeries]);

  if (!settings || !updateSettings) return null;

  return (
    <>
      <SelectFilter
        enableSearchFilter={false}
        label="Metric Name"
        showSearch={false}
        style={{ width: 220 }}
        value={settings.name}
        onChange={handleChangeName}>
        {systemSeries &&
          Object.keys(systemSeries).map((name) => (
            <Option key={name} value={name}>
              {name}
            </Option>
          ))}
      </SelectFilter>
      <SelectFilter
        enableSearchFilter={false}
        label="Agent Name"
        showSearch={false}
        style={{ width: 220 }}
        value={settings.agentId}
        onChange={handleChangeAgentId}>
        {settings.name &&
          systemSeries &&
          Object.keys(systemSeries[settings.name]).map((agentId) => (
            <Option key={agentId} value={agentId}>
              {agentId}
            </Option>
          ))}
      </SelectFilter>
      {uuidOptions.length !== 0 && (
        <SelectFilter
          allowClear={true}
          enableSearchFilter={false}
          label="GPU"
          placeholder="All"
          showSearch={false}
          style={{ width: 220 }}
          value={settings.gpuUuid}
          onChange={handleChangeGpuUuid}>
          {uuidOptions.map((gpuUuid) => (
            <Option key={gpuUuid} value={gpuUuid}>
              {gpuUuid}
            </Option>
          ))}
        </SelectFilter>
      )}
    </>
  );
};

export default SystemMetricFilter;
