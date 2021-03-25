import { Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useEffect } from 'react';

import SelectFilter from 'components/SelectFilter';
import { TrialDetails } from 'types';

import { MetricType, useFetchAvailableSeries } from './utils';

const { Option } = Select;

export interface FiltersInterface {
  agentId?: string;
  gpuUuid?: string;
  name?: string;
}

export interface Props {
  onChange: (newValue: FiltersInterface) => void,
  trial: TrialDetails;
  value: FiltersInterface;
}

const SystemMetricFilter: React.FC<Props> = ({ onChange, trial, value }: Props) => {
  const systemSeries = useFetchAvailableSeries(trial.id)[MetricType.System];

  /*
   * Set default value value when available series has loaded.
   */
  useEffect(() => {
    if (!systemSeries) return;
    if (value.agentId && value.name) return;

    const newFilters: FiltersInterface = {
      agentId: value.agentId,
      name: value.name,
    };

    if (!value.name) {
      if (Object.keys(systemSeries).includes('gpu_util')) newFilters.name = 'gpu_util';
      else if (Object.keys(systemSeries).includes('cpu_util')) newFilters.name = 'cpu_util';
      else newFilters.name = Object.keys(systemSeries)[0];
    }

    if (!value.agentId) {
      newFilters.agentId = Object.keys(systemSeries[newFilters.name as unknown as string])[0];
    }

    onChange(newFilters);
  }, [ onChange, systemSeries, value.agentId, value.name ]);

  const handleChangeAgentId = useCallback((newAgentId: SelectValue) => {
    onChange({ agentId: newAgentId as unknown as string, name: value.name });
  }, [ value.name, onChange ]);
  const handleChangeGpuUuid = useCallback((newGpuUuid: SelectValue) => {
    onChange({
      agentId: value.agentId,
      gpuUuid: newGpuUuid as unknown as string,
      name: value.name,
    });
  }, [ value.agentId, value.name, onChange ]);
  const handleChangeName = useCallback((newName: SelectValue) => {
    onChange({ name: newName as unknown as string });
  }, [ onChange ]);

  return (
    <>

      <SelectFilter
        enableSearchFilter={false}
        label='Metric Name'
        showSearch={false}
        style={{ width: 150 }}
        value={value.name}
        onChange={handleChangeName}
      >
        {systemSeries && Object.keys(systemSeries).map(name => (
          <Option key={name} value={name}>{name}</Option>
        ))}
      </SelectFilter>

      <SelectFilter
        enableSearchFilter={false}
        label='Agent Name'
        showSearch={false}
        style={{ width: 150 }}
        value={value.agentId}
        onChange={handleChangeAgentId}
      >
        {value.name && Object.keys(systemSeries[value.name]).map(agentId => (
          <Option key={agentId} value={agentId}>{agentId}</Option>
        ))}
      </SelectFilter>

      {value.name && value.agentId && systemSeries[value.name][value.agentId].length > 0 && (
        <SelectFilter
          allowClear={true}
          enableSearchFilter={false}
          label='GPU'
          placeholder='All'
          showSearch={false}
          style={{ width: 150 }}
          value={value.gpuUuid}
          onChange={handleChangeGpuUuid}
        >
          {systemSeries[value.name][value.agentId].map(gpuUuid => (
            <Option key={gpuUuid} value={gpuUuid}>{gpuUuid}</Option>
          ))}
        </SelectFilter>
      )}

    </>
  );
};

export default SystemMetricFilter;
