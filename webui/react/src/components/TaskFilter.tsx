import { Select, Tooltip } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useMemo } from 'react';
import styled from 'styled-components';
import { theme } from 'styled-tools';

import Icon from 'components/Icon';
import LayoutHelper from 'components/LayoutHelper';
import { ShirtSize } from 'themes';
import { CommandState, RecentTask, RunState, TaskType, User } from 'types';
import { isNumber } from 'utils/data';
import { commandStateToLabel, runStateToLabel } from 'utils/types';

const { Option, OptGroup } = Select;

export interface TaskFilters {
  limit: number;
  states: string[];
  userId?: number;
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

const TaskFilter: React.FC<Props> = (props: Props) => {
  const handleTypeClick = (taskType: TaskType): (() => void) => {
    return useCallback((): void => {
      const types = { ...props.filters.types };
      types[taskType] = !props.filters.types[taskType];
      props.onChange({ ...props.filters, types });
    }, [ props.filters.types, props.onChange ]);
  };

  const handleStateSelect = useCallback((value: SelectValue): void => {
    if (typeof value !== 'string') return;
    const key = value.toUpperCase();
    props.onChange({ ...props.filters, states: [ key ] });
  }, [ props.filters, props.onChange ]);

  const handleUserFilter = useCallback((search: string, option) => {
    return option.props.children.indexOf(search) !== -1;
  }, []);

  const handleUserSelect = useCallback((value: SelectValue) => {
    const userId = isNumber(value) ? value as number : undefined;
    props.onChange({ ...props.filters, userId });
  }, [ props.filters.userId ]);

  const handleLimitSelect = useCallback((limit: number): void => {
    props.onChange({ ...props.filters, limit });
  }, [ props.filters, props.onChange ]);

  const filterTaskButtons = taskTypeOrder.map(info => (
    <Tooltip key={info.label} placement="top" title={info.label}>
      <FilterButton
        aria-label={info.label}
        className={props.filters.types[info.type] ? 'active' : ''}
        onClick={handleTypeClick(info.type)}>
        <Icon name={info.type.toLocaleLowerCase()} />
      </FilterButton>
    </Tooltip>
  ));

  const runStateOptions = useMemo(() => Object.values(RunState).map((value) => {
    return <Option key={value} value={value}>{runStateToLabel[value]}</Option>;
  }), [ Option, RunState, runStateToLabel ]);

  const commandStateOptions = useMemo(() => Object.values(CommandState).map((value) => {
    return <Option key={value} value={value}>{commandStateToLabel[value]}</Option>;
  }), [ Option, RunState, commandStateToLabel ]);

  const defaultUserId = useMemo((): number | string => {
    const userId = props.filters.userId;
    const userIds = props.users.map(user => user.id);
    if (userId && userIds.includes(userId)) return userId;
    return 'all';
  }, [ props.filters.userId, props.users ]);

  return (
    <LayoutHelper gap={ShirtSize.jumbo} yCenter>
      <LayoutHelper gap={ShirtSize.medium}>{filterTaskButtons}</LayoutHelper>
      <div>
        <Label>State</Label>
        <Select
          defaultValue={props.filters.states[0]}
          dropdownMatchSelectWidth={false}
          onSelect={handleStateSelect}>
          <Option key="all" value="ALL">All</Option>
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
          defaultValue={defaultUserId}
          dropdownMatchSelectWidth={false}
          filterOption={handleUserFilter}
          optionFilterProp="children"
          showSearch={true}
          style={{ width: '10rem' }}
          onSelect={handleUserSelect}>
          <Option key="all" value="all">All</Option>
          {props.users.map(user => (
            <Option key={user.id} value={user.id}>{user.username}</Option>
          ))}
        </Select>
      </div>
      <div>
        <Label>Limit</Label>
        <Select
          defaultValue={props.filters.limit}
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
  if (states[0] === 'ALL') return true;

  const targetStateRun = states[0] as RunState;
  const targetStateCmd = states[0] as CommandState;

  return [ targetStateRun, targetStateCmd ].includes(task.state);
};

export const filterTasks = (tasks: RecentTask[], filters: TaskFilters): RecentTask[] => {
  const isAllTypes = !Object.values(filters.types).includes(true);
  return tasks
    .filter(task => !task.archived)
    .filter(task => matchesState(task, filters.states))
    .filter(task => isAllTypes || filters.types[task.type])
    .slice(0, filters.limit);
};
