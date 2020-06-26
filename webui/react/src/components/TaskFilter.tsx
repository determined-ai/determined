import { Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useMemo } from 'react';

import { AnyTask, CommandState, CommandTask, CommandType, RunState, TaskType, User } from 'types';
import { capitalize } from 'utils/string';
import { isExperimentTask } from 'utils/task';

import IconFilterButtons from './IconFilterButtons';
import SelectFilter, { ALL_VALUE } from './SelectFilter';
import StateSelectFilter from './StateSelectFilter';
import css from './TaskFilter.module.scss';
import UserSelectFilter from './UserSelectFilter';

const { Option } = Select;

export { ALL_VALUE };

export interface TaskFilters<T extends TaskType = TaskType> {
  limit: number;
  states: string[];
  username?: string;
  types: Record<T, boolean>;
}

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
  filters, onChange, showExperiments = true, showLimit = true,
}: Props<T>) => {
  const allTypesOff = !Object.values(filters.types).reduce((acc, type) => (acc || type), false);
  const showCommandStates = allTypesOff ||
    filters.types[CommandType.Command as T] || filters.types[CommandType.Notebook as T] ||
    filters.types[CommandType.Shell as T] || filters.types[CommandType.Tensorboard as T];
  const showExperimentStates = showExperiments && (allTypesOff || filters.types['Experiment' as T]);

  const handleTypeClick = useCallback((id: string) => {
    const types = { ...filters.types, [id as T]: !filters.types[id as T] };
    onChange({ ...filters, types });
  }, [ filters, onChange ]);

  const handleStateChange = useCallback((value: SelectValue): void => {
    if (typeof value !== 'string') return;
    onChange({ ...filters, states: [ value ] });
  }, [ filters, onChange ]);

  const handleUserChange = useCallback((value: SelectValue) => {
    const username = value === ALL_VALUE ? undefined : value as string;
    onChange({ ...filters, username });
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
      active: filters.types[config.id as T],
      icon: config.id.toLocaleLowerCase(),
      id: config.id,
      label: capitalize(config.id),
    }));
  }, [ filters.types, showExperiments ]);

  return (
    <div className={css.base}>
      <IconFilterButtons buttons={filterTypeConfig} onClick={handleTypeClick} />
      <StateSelectFilter
        showCommandStates={showCommandStates}
        showExperimentStates={showExperimentStates}
        value={filters.states}
        onChange={handleStateChange} />
      <UserSelectFilter value={filters.username} onChange={handleUserChange} />
      {showLimit && <SelectFilter
        label="Limit"
        showSearch={false}
        value={filters.limit}
        onSelect={handleLimitSelect}>
        {limitOptions.map(limit => <Option key={limit} value={limit}>{limit}</Option>)}
      </SelectFilter>}
    </div>
  );
};

export default TaskFilter;

const matchesState = <T extends AnyTask>(task: T, states: string[]): boolean => {
  if (states[0] === ALL_VALUE) return true;

  const targetStateRun = states[0] as RunState;
  const targetStateCmd = states[0] as CommandState;

  return [ targetStateRun, targetStateCmd ].includes(task.state);
};

const matchesUser = <T extends AnyTask>(task: T, users: User[], username?: string): boolean => {
  if (!username) return true;
  const selectedUser = users.find(u => u.username === username);
  return !!selectedUser && (task.ownerId === selectedUser.id);
};

export const filterTasks = <T extends TaskType = TaskType, A extends AnyTask = AnyTask>(
  tasks: A[], filters: TaskFilters<T>, users: User[],
): A[] => {
  const isAllTypes = !Object.values(filters.types).includes(true);
  return tasks
    .filter(task => matchesUser<A>(task as A, users, filters.username))
    .filter(task => {
      if (!isExperimentTask(task)) return true;
      return !task.archived;
    })
    .filter(task => matchesState<A>(task as A, filters.states))
    .filter(task => {
      const type = isExperimentTask(task) ? 'Experiment' : (task as CommandTask).type;
      return isAllTypes || filters.types[type as T];
    })
    .slice(0, filters.limit);
};
