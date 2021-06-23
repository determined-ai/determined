import { Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useMemo } from 'react';

import SelectFilter from 'components/SelectFilter';
import { useProfilesFilterContext } from 'pages/TrialDetails/Profiles/ProfilesFiltersProvider';

const { Option } = Select;

export interface FiltersInterface {
  agentId?: string;
  gpuUuid?: string;
  name?: string;
}

const SystemMetricFilter: React.FC = () => {
  const { filters, setFilters, systemSeries } = useProfilesFilterContext();

  const handleChangeAgentId = useCallback((newAgentId: SelectValue) => {
    setFilters({ agentId: newAgentId as unknown as string, name: filters.name });
  }, [ filters.name, setFilters ]);
  const handleChangeGpuUuid = useCallback((newGpuUuid: SelectValue) => {
    setFilters({
      agentId: filters.agentId,
      gpuUuid: newGpuUuid as unknown as string,
      name: filters.name,
    });
  }, [ filters.agentId, filters.name, setFilters ]);
  const handleChangeName = useCallback((newName: SelectValue) => {
    setFilters({ name: newName as unknown as string });
  }, [ setFilters ]);

  const uuidOptions = useMemo(() => {
    return filters.name && filters.agentId &&
      systemSeries[filters.name][filters.agentId].filter(uuid => uuid && uuid.length > 0) || [];
  }, [ systemSeries, filters ]);

  return (
    <>

      <SelectFilter
        enableSearchFilter={false}
        label="Metric Name"
        showSearch={false}
        style={{ width: 220 }}
        value={filters.name}
        onChange={handleChangeName}
      >
        {systemSeries && Object.keys(systemSeries).map(name => (
          <Option key={name} value={name}>{name}</Option>
        ))}
      </SelectFilter>

      <SelectFilter
        enableSearchFilter={false}
        label="Agent Name"
        showSearch={false}
        style={{ width: 220 }}
        value={filters.agentId}
        onChange={handleChangeAgentId}
      >
        {filters.name && Object.keys(systemSeries[filters.name]).map(agentId => (
          <Option key={agentId} value={agentId}>{agentId}</Option>
        ))}
      </SelectFilter>

      { uuidOptions.length > 0 &&
        (<SelectFilter
          allowClear={true}
          enableSearchFilter={false}
          label="GPU"
          placeholder="All"
          showSearch={false}
          style={{ width: 220 }}
          value={filters.gpuUuid}
          onChange={handleChangeGpuUuid}
        >
          {uuidOptions.map(gpuUuid => (
            <Option key={gpuUuid} value={gpuUuid}>{gpuUuid}</Option>
          ))}
        </SelectFilter>
        )}
    </>
  );
};

export default SystemMetricFilter;
