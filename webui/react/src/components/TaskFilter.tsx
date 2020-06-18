import { Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useMemo } from 'react';

import Icon from 'components/Icon';
import { CommandState, CommandType, RecentTask, RunState, TaskType, User } from 'types';
import { capitalize } from 'utils/string';
import { isExperimentTask } from 'utils/task';
import { commandStateToLabel, runStateToLabel } from 'utils/types';

import IconFilterButtons from './IconFilterButtons';
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

const taskTypeConfig = [
  { id: 'Experiment' },
  { id: CommandType.Notebook },
  { id: CommandType.Tensorboard },
  { id: CommandType.Shell },
  { id: CommandType.Command },
];

const selectIcon = <Icon name="arrow-down" size="tiny" />;

const TaskFilter: React.FC<Props> = ({ authUser, filters, onChange, users }: Props) => {
  const handleTypeClick = useCallback((id: string) => {
    const idAsType = id as TaskType;
    const types = { ...filters.types, [idAsType]: !filters.types[idAsType] };
    onChange({ ...filters, types });
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

  const filterTypeConfig = useMemo(() => {
    return taskTypeConfig.map(config => ({
      active: filters.types[config.id as TaskType],
      icon: config.id.toLocaleLowerCase(),
      id: config.id,
      label: capitalize(config.id),
    }));
  }, [ filters.types ]);

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
      <IconFilterButtons buttons={filterTypeConfig} onClick={handleTypeClick} />
      <div className={css.filter}>
        <div className={css.label}>State</div>
        <Select
          defaultValue={filters.states[0]}
          dropdownMatchSelectWidth={false}
          suffixIcon={selectIcon}
          onSelect={handleStateSelect}>
          <Option key={ALL_VALUE} value={ALL_VALUE}>All</Option>
          <OptGroup key="expGroup" label="Experiment States">
            {Object.values(RunState).map((value) => (
              <Option key={value} value={value}>{runStateToLabel[value]}</Option>
            ))}
          </OptGroup>
          <OptGroup key="cmdGroup" label="Command States">
            {Object.values(CommandState).map((value) => (
              <Option key={value} value={value}>{commandStateToLabel[value]}</Option>
            ))}
          </OptGroup>
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

const matchesState = (task: RecentTask, states: string[]): boolean => {
  if (states[0] === ALL_VALUE) return true;

  const targetStateRun = states[0] as RunState;
  const targetStateCmd = states[0] as CommandState;

  return [ targetStateRun, targetStateCmd ].includes(task.state);
};

const matchesUser = (task: RecentTask, users: User[], username?: string): boolean => {
  if (!username) return true;
  const selectedUser = users.find(u => u.username === username);
  return !!selectedUser && (task.ownerId === selectedUser.id);
};

export const filterTasks =
  (tasks: RecentTask[], filters: TaskFilters, users: User[]): RecentTask[] => {
    const isAllTypes = !Object.values(filters.types).includes(true);
    return tasks
      .filter(task => matchesUser(task, users, filters.username))
      .filter(task => {
        if (!isExperimentTask(task)) return true;
        return !task.archived;
      })
      .filter(task => matchesState(task, filters.states))
      .filter(task => {
        const type = isExperimentTask(task) ? 'Experiment' : task.type;
        return isAllTypes || filters.types[type];
      })
      .slice(0, filters.limit);
  };
