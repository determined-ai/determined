import { Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback } from 'react';
import styled from 'styled-components';
import { theme } from 'styled-tools';

import IconCounter, { IconCounterType } from 'components/IconCounter';
import LayoutHelper from 'components/LayoutHelper';
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
  counts: {[key in TaskType]: number};
  users: User[];
}

const TaskFilter: React.FC<Props> = (props: Props) => {
  const handleTypeClick = (taskType: TaskType): (() => void) => {
    return useCallback((): void => {
      const types = { ...props.filters.types };
      types[taskType] = !props.filters.types[taskType];
      props.onChange({ ...props.filters, types });
    }, [ props.filters, props.onChange ]);
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
  }, [ props.filters, props.onChange ]);

  const handleLimitSelect = useCallback((value: number): void => {
    props.onChange({ ...props.filters, limit: value });
  }, [ props.filters, props.onChange ]);

  const RunStateOptions = Object.values(RunState).map((val) => {
    return <Option key={val} value={val}>{runStateToLabel[val]}</Option>;
  });

  const CmdStateOptions = Object.values(CommandState).map((val) => {
    return <Option key={val} value={val}>{commandStateToLabel[val]}</Option>;
  });

  const stateOptions = [
    <Option key="all" value="ALL">All</Option>,
    <OptGroup key="expGroup" label="Experiment States">{RunStateOptions}</OptGroup>,
    <OptGroup key="cmdGroup" label="Command States">{CmdStateOptions}</OptGroup>,
  ];

  /* eslint-disable comma-dangle */
  const stateSelector = (
    <LabeledInput>
      <label>State</label>
      <Select
        defaultValue={props.filters.states[0]}
        dropdownMatchSelectWidth={false} onSelect={handleStateSelect}>
        {stateOptions}
      </Select>
    </LabeledInput>
  );
  /* eslint-enable comma-dangle */

  const limitPicker = (
    <LabeledInput>
      <label>Limit</label>
      <Select defaultValue={props.filters.limit} onSelect={handleLimitSelect}>
        <Option value={10}>10</Option>
        <Option value={25}>25</Option>
        <Option value={50}>50</Option>
      </Select>
    </LabeledInput>
  );

  const getIconType = (taskType: TaskType): IconCounterType => {
    return props.filters.types[taskType] ? IconCounterType.Active : IconCounterType.Disabled;
  };

  const typeFilters =
    ((): React.ReactNode[] => {
      const taskTypes = [
        TaskType.Experiment,
        TaskType.Notebook,
        TaskType.Tensorboard,
        TaskType.Shell,
        TaskType.Command,
      ];
      return taskTypes.map((tType, idx) => (
        <IconCounter
          count={props.counts[tType]}
          key={idx}
          name={tType.toLowerCase()}
          type={getIconType(tType)}
          onClick={handleTypeClick(tType)} />
      ));
    })();

  return (
    <LayoutHelper gap="jumbo" yCenter>
      <LayoutHelper gap="big">
        {typeFilters}
      </LayoutHelper>
      {stateSelector}
      <LabeledInput>
        <label>Users</label>
        <Select
          defaultValue={props.filters.userId || 'all'}
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
      </LabeledInput>
      {limitPicker}
    </LayoutHelper>
  );
};

const LabeledInput = styled.div`
  & > label {
    font-size: ${theme('sizes.font.medium')};
    font-weight: bold;
    margin-right: ${theme('sizes.layout.medium')};
  }
`;

export default TaskFilter;

const matchesState = (task: RecentTask, states: string[]): boolean =>  {
  if (states[0] === 'ALL') return true;

  const targetStateRun = states[0] as RunState;
  const targetStateCmd = states[0] as CommandState;

  return [ targetStateRun, targetStateCmd ].indexOf(task.state) !== -1;
};

export const filterTasks = (tasks: RecentTask[], filters: TaskFilters): RecentTask[] => {
  return tasks
    .filter(task => matchesState(task, filters.states))
    .filter(task => filters.types[task.type])
    .filter(task => !task.archived)
    .filter(task => !filters.userId || task.ownerId === filters.userId)
    .slice(0, filters.limit);
};

export const getTaskCounts = (tasks: RecentTask[]): {[key in TaskType]: number} => {
  return tasks.reduce((acc, task) => {
    acc[task.type]++;
    return acc;
  }, {
    [TaskType.Command]: 0,
    [TaskType.Experiment]: 0,
    [TaskType.Notebook]: 0,
    [TaskType.Tensorboard]: 0,
    [TaskType.Shell]: 0,
  });
};
