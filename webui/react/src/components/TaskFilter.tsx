import { Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useMemo } from 'react';

import { ALL_VALUE, CommandType, TaskFilters, TaskType } from 'types';
import { capitalize } from 'utils/string';

import IconFilterButtons from './IconFilterButtons';
import ResponsiveFilters from './ResponsiveFilters';
import SelectFilter from './SelectFilter';
import StateSelectFilter from './StateSelectFilter';
import UserSelectFilter from './UserSelectFilter';

const { Option } = Select;

interface Props<T extends TaskType> {
  filters: TaskFilters<T>;
  onChange: (filters: TaskFilters<T>) => void;
  showExperiments?: boolean;
  showLimit?: boolean;
}

type TaskFilterFC = <T extends TaskType = TaskType>(props: Props<T>) => React.ReactElement;

const limitOptions: number[] = [ 10, 25, 50 ];

const commandConfig = [
  { id: CommandType.Notebook },
  { id: CommandType.Tensorboard },
  { id: CommandType.Shell },
  { id: CommandType.Command },
];
const experimentConfig = [ { id: 'Experiment' } ];

const TaskFilter: TaskFilterFC = <T extends TaskType = TaskType>({
  filters, onChange,
  showExperiments = true,
  showLimit = true,
}: Props<T>) => {
  const handleTypeClick = useCallback((id: string) => {
    const typeId = id as T;
    const types = filters.types ? [ ...filters.types ] : [];
    const index = types.indexOf(typeId);
    if (index === -1) types.push(typeId);
    else types.splice(index, 1);
    onChange({ ...filters, types: types.length === 0 ? undefined : types });
  }, [ filters, onChange ]);

  const handleStateChange = useCallback((value: SelectValue): void => {
    if (typeof value !== 'string') return;
    onChange({ ...filters, states: [ value ] });
  }, [ filters, onChange ]);

  const handleUserChange = useCallback((value: SelectValue) => {
    const users = value === ALL_VALUE ? undefined : [ value as string ];
    onChange({ ...filters, users });
  }, [ filters, onChange ]);

  const handleLimitSelect = useCallback((limit: SelectValue): void => {
    onChange({ ...filters, limit: limit as number });
  }, [ filters, onChange ]);

  const filterTypeConfig = useMemo(() => {
    const taskTypeConfig = [
      ...(showExperiments ? experimentConfig : []),
      ...commandConfig,
    ];
    return taskTypeConfig.map(config => ({
      active: Array.isArray(filters.types) && filters.types.includes(config.id as T),
      icon: config.id.toLocaleLowerCase(),
      id: config.id,
      label: capitalize(config.id),
    }));
  }, [ filters.types, showExperiments ]);

  return (
    <ResponsiveFilters hasFiltersApplied={false}>
      <IconFilterButtons buttons={filterTypeConfig} onClick={handleTypeClick} />
      <StateSelectFilter
        showCommandStates={true}
        showExperimentStates={showExperiments}
        value={filters.states}
        onChange={handleStateChange} />
      <UserSelectFilter value={filters.users} onChange={handleUserChange} />
      {showLimit && <SelectFilter
        label="Limit"
        showSearch={false}
        value={filters.limit}
        onSelect={handleLimitSelect}>
        {limitOptions.map(limit => <Option key={limit} value={limit}>{limit}</Option>)}
      </SelectFilter>}
    </ResponsiveFilters>
  );
};

export default TaskFilter;
