import { Select as AntdSelect } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useMemo } from 'react';

import Select from 'components/kit/Select';
const { Option } = AntdSelect;
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
  updateSettings?: UpdateSettings;
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

  const validAgentIds =
    settings.name && systemSeries ? Object.keys(systemSeries[settings.name]) : [];

  return (
    <>
      <Select
        enableSearchFilter={false}
        label="Metric Name"
        value={settings.name}
        onChange={handleChangeName}>
        {systemSeries &&
          Object.keys(systemSeries).map((name) => (
            <Option key={name} value={name}>
              {name}
            </Option>
          ))}
      </Select>
      <Select
        enableSearchFilter={false}
        label="Agent Name"
        value={validAgentIds.includes(settings.agentId as string) ? settings.agentId : undefined}
        onChange={handleChangeAgentId}>
        {validAgentIds.map((agentId) => (
          <Option key={agentId} value={agentId}>
            {agentId}
          </Option>
        ))}
      </Select>
      {uuidOptions.length !== 0 && (
        <Select
          allowClear={true}
          enableSearchFilter={false}
          label="GPU"
          placeholder="All"
          value={settings.gpuUuid}
          onChange={handleChangeGpuUuid}>
          {uuidOptions.map((gpuUuid) => (
            <Option key={gpuUuid} value={gpuUuid}>
              {gpuUuid}
            </Option>
          ))}
        </Select>
      )}
    </>
  );
};

export default SystemMetricFilter;
