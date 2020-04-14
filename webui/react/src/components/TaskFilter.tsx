import { Select, Tooltip } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useMemo } from 'react';
import styled from 'styled-components';
import { theme } from 'styled-tools';

import Icon from 'components/Icon';
import LayoutHelper from 'components/LayoutHelper';
import { ShirtSize } from 'themes';
import { CommandState, RecentTask, RunState, TaskType, User } from 'types';
import { commandStateToLabel, runStateToLabel } from 'utils/types';

const { Option, OptGroup } = Select;

export const ALL_VALUE = 'all';

export interface TaskFilters {
  limit: number;
  states: string[];
  username?: string;
  types: Record<TaskType, boolean>;
}

interface Props {
  filters: TaskFilters;
  onChange: (filters: TaskFilters) => void;
  users: User[];
}

const limitOptions: number[] = [ 10, 25, 50 ];

const taskTypeOrder = [
  { label: 'Experiments', type: TaskType.Experiment },
  { label: 'Notebooks', type: TaskType.Notebook },
  { label: 'TensorBoards', type: TaskType.Tensorboard },
  { label: 'Shells', type: TaskType.Shell },
  { label: 'Commands', type: TaskType.Command },
];

const TaskFilter: React.FC<Props> = ({ filters, onChange, users }: Props) => {
  const handleTypeClick = useCallback((taskType: TaskType): (() => void) => {
    return (): void => {
      const types = { ...filters.types };
      types[taskType] = !filters.types[taskType];
      onChange({ ...filters, types });
    };
  }, [ filters, onChange ]);

  const handleStateSelect = useCallback((value: SelectValue): void => {
    if (typeof value !== 'string') return;
    onChange({ ...filters, states: [ value ] });
  }, [ filters, onChange ]);

  const handleUserFilter = useCallback((search: string, option) => {
    return option.props.children.indexOf(search) !== -1;
  }, []);

  const handleUserSelect = useCallback((value: SelectValue) => {
    const username = value === ALL_VALUE ? undefined : value as string;
    onChange({ ...filters, username });
  }, [ filters, onChange ]);

  const handleLimitSelect = useCallback((limit: number): void => {
    onChange({ ...filters, limit });
  }, [ filters, onChange ]);

  const selectIcon = <Icon name="arrow-down" size="small" />;

  const filterTaskButtons = taskTypeOrder.map(info => (
    <Tooltip key={info.label} placement="top" title={info.label}>
      <FilterButton
        aria-label={info.label}
        className={filters.types[info.type] ? 'active' : ''}
        onClick={handleTypeClick(info.type)}>
        <Icon name={info.type.toLocaleLowerCase()} />
      </FilterButton>
    </Tooltip>
  ));

  const runStateOptions = useMemo(() => Object.values(RunState).map((value) => {
    return <Option key={value} value={value}>{runStateToLabel[value]}</Option>;
  }), [ ]);

  const commandStateOptions = useMemo(() => Object.values(CommandState).map((value) => {
    return <Option key={value} value={value}>{commandStateToLabel[value]}</Option>;
  }), [ ]);

  const usernameFilter = filters.username;
  const defaultUsername = useMemo((): number | string => {
    return usernameFilter || ALL_VALUE;
  }, [ usernameFilter ]);

  return (
    <LayoutHelper gap={ShirtSize.jumbo} yCenter>
      <LayoutHelper gap={ShirtSize.medium}>{filterTaskButtons}</LayoutHelper>
      <div>
        <Label>State</Label>
        <Select
          defaultValue={filters.states[0]}
          dropdownMatchSelectWidth={false}
          suffixIcon={selectIcon}
          onSelect={handleStateSelect}>
          <Option key={ALL_VALUE} value={ALL_VALUE}>All</Option>
          <OptGroup key="expGroup" label="Experiment States">
            {runStateOptions}
          </OptGroup>
          <OptGroup key="cmdGroup" label="Command States">
            {commandStateOptions}
          </OptGroup>
        </Select>
      </div>
      <div>
        <Label>Users</Label>
        <Select
          defaultValue={defaultUsername}
          dropdownMatchSelectWidth={false}
          filterOption={handleUserFilter}
          optionFilterProp="children"
          showSearch={true}
          style={{ width: '10rem' }}
          suffixIcon={selectIcon}
          onSelect={handleUserSelect}>
          <Option key={ALL_VALUE} value={ALL_VALUE}>All</Option>
          {users.map(user => (
            <Option key={user.id} value={user.username}>{user.username}</Option>
          ))}
        </Select>
      </div>
      <div>
        <Label>Limit</Label>
        <Select
          defaultValue={filters.limit}
          suffixIcon={selectIcon}
          onSelect={handleLimitSelect}>
          {limitOptions.map(limit => <Option key={limit} value={limit}>{limit}</Option>)}
        </Select>
      </div>
    </LayoutHelper>
  );
};

const FilterButton = styled.div`
  align-items: center;
  background-color: ${theme('colors.monochrome.17')};
  border: solid ${theme('sizes.border.width')} ${theme('colors.monochrome.12')};
  border-radius: ${theme('sizes.border.radius')};
  cursor: pointer;
  display: flex;
  height: ${theme('sizes.layout.huge')};
  padding: 0 ${theme('sizes.layout.small')};
  transition: 0.2s;
  &.active {
    border-color: ${theme('colors.states.active')};
    color: ${theme('colors.states.active')};
  }
  &:hover { border-color: ${theme('colors.states.active')}; }
`;

const Label = styled.label`
  font-size: ${theme('sizes.font.medium')};
  font-weight: bold;
  margin-right: ${theme('sizes.layout.medium')};
`;

export default TaskFilter;

const matchesState = (task: RecentTask, states: string[]): boolean =>  {
  if (states[0] === ALL_VALUE) return true;

  const targetStateRun = states[0] as RunState;
  const targetStateCmd = states[0] as CommandState;

  return [ targetStateRun, targetStateCmd ].includes(task.state);
};

const matchesUser = (task: RecentTask, users: User[], username?: string): boolean =>  {
  if (!username) return true;
  const selectedUser = users.find(u => u.username === username);
  return !!selectedUser && (task.ownerId === selectedUser.id);
};

export const filterTasks =
  (tasks: RecentTask[], filters: TaskFilters, users: User[]): RecentTask[] => {
    const isAllTypes = !Object.values(filters.types).includes(true);
    return tasks
      .filter(task => matchesUser(task, users, filters.username))
      .filter(task => !task.archived)
      .filter(task => matchesState(task, filters.states))
      .filter(task => isAllTypes || filters.types[task.type])
      .slice(0, filters.limit);
  };
