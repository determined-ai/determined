import { Select, Tooltip } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback } from 'react';

import Icon from 'components/Icon';
import { CommandState, RecentTask, RunState, TaskType, User } from 'types';
import { commandStateToLabel, runStateToLabel } from 'utils/types';

import css from './TaskFilter.module.scss';

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
  authUser?: User;
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

const TaskFilter: React.FC<Props> = ({ authUser, filters, onChange, users }: Props) => {
  const handleTypeClick = useCallback((taskType: TaskType): (() => void) => {
    return (): void => {
      const types = { ...filters.types, [taskType]: !filters.types[taskType] };
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

  const selectIcon = <Icon name="arrow-down" size="tiny" />;

  const filterTypeButtons = taskTypeOrder.map(info => {
    const typeButtonClasses = [ css.typeButton ];
    if (filters.types[info.type]) typeButtonClasses.push(css.active);
    return (
      <Tooltip key={info.label} placement="top" title={info.label}>
        <button aria-label={info.label}
          className={typeButtonClasses.join(' ')}
          tabIndex={0}
          onClick={handleTypeClick(info.type)}>
          <Icon name={info.type.toLocaleLowerCase()} />
        </button>
      </Tooltip>
    );
  });

  const runStateOptions = Object.values(RunState).map((value) => {
    return <Option key={value} value={value}>{runStateToLabel[value]}</Option>;
  });

  const commandStateOptions = Object.values(CommandState).map((value) => {
    return <Option key={value} value={value}>{commandStateToLabel[value]}</Option>;
  });

  const userToSelectOption = (user: User): React.ReactNode =>
    <Option key={user.id} value={user.username}>{user.username}</Option>;

  const userOptions = (): React.ReactNode[] => {
    const options: React.ReactNode[] = [ <Option key={ALL_VALUE} value={ALL_VALUE}>All</Option> ];
    if (authUser) {
      options.push(userToSelectOption(authUser));
    }
    const restOfOptions = users
      .filter(u => (!authUser || u.id !== authUser.id))
      .sort((a, b) => a.username.localeCompare(b.username, 'en'))
      .map(userToSelectOption);
    options.push(...restOfOptions);

    return options;
  };

  return (
    <div className={css.base}>
      <div className={css.typeButtons}>{filterTypeButtons}</div>
      <div className={css.filter}>
        <div className={css.label}>State</div>
        <Select
          defaultValue={filters.states[0]}
          dropdownMatchSelectWidth={false}
          suffixIcon={selectIcon}
          onSelect={handleStateSelect}>
          <Option key={ALL_VALUE} value={ALL_VALUE}>All</Option>
          <OptGroup key="expGroup" label="Experiment States">{runStateOptions}</OptGroup>
          <OptGroup key="cmdGroup" label="Command States">{commandStateOptions}</OptGroup>
        </Select>
      </div>
      <div className={css.filter}>
        <div className={css.label}>Users</div>
        <Select
          defaultValue={filters.username || ALL_VALUE}
          dropdownMatchSelectWidth={false}
          filterOption={handleUserFilter}
          optionFilterProp="children"
          showSearch={true}
          style={{ width: '10rem' }}
          suffixIcon={selectIcon}
          onSelect={handleUserSelect}>
          {userOptions()}
        </Select>
      </div>
      <div className={css.filter}>
        <div className={css.label}>Limit</div>
        <Select
          defaultValue={filters.limit}
          suffixIcon={selectIcon}
          onSelect={handleLimitSelect}>
          {limitOptions.map(limit => <Option key={limit} value={limit}>{limit}</Option>)}
        </Select>
      </div>
    </div>
  );
};

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
