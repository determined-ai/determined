import { Input, Table } from 'antd';
import React, { useCallback, useMemo, useState } from 'react';

import Icon from 'components/Icon';
import { makeClickHandler } from 'components/Link';
import linkCss from 'components/Link.module.scss';
import Page from 'components/Page';
import TaskFilter from 'components/TaskFilter';
import Auth from 'contexts/Auth';
import { Commands, Notebooks, Shells, Tensorboards } from 'contexts/Commands';
import Users from 'contexts/Users';
import useStorage from 'hooks/useStorage';
import { ALL_VALUE, CommandTask, CommandType, TaskFilters } from 'types';
import { canBeOpened, filterTasks } from 'utils/task';
import { commandToTask } from 'utils/types';

import css from './TaskList.module.scss';
import { columns } from './TaskList.table';

const defaultFilters: TaskFilters<CommandType> = {
  limit: 25,
  states: [ ALL_VALUE ],
  types: {
    [CommandType.Command]: false,
    [CommandType.Notebook]: false,
    [CommandType.Shell]: false,
    [CommandType.Tensorboard]: false,
  },
  username: undefined,
};

const TaskList: React.FC = () => {
  const auth = Auth.useStateContext();
  const users = Users.useStateContext();
  const commands = Commands.useStateContext();
  const notebooks = Notebooks.useStateContext();
  const shells = Shells.useStateContext();
  const tensorboards = Tensorboards.useStateContext();
  const storage = useStorage('tasklist');
  const initFilters = storage.getWithDefault('filters',
    { ...defaultFilters, username: (auth.user || {}).username });
  const [ filters, setFilters ] = useState<TaskFilters<CommandType>>(initFilters);
  const [ search, setSearch ] = useState('');

  const sources = [
    commands,
    notebooks,
    shells,
    tensorboards,
  ];

  const loadedTasks = sources
    .filter(src => src.data !== undefined)
    .map(src => src.data || [])
    .reduce((acc, cur) => [ ...acc, ...cur ], [])
    .map(commandToTask);

  const hasLoaded = sources.find(src => src.hasLoaded);

  const filteredTasks = useMemo(() => {
    return filterTasks(loadedTasks, filters, users.data || [], search);
  }, [ filters, loadedTasks, search, users.data ]);

  const handleSearchChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setSearch(e.target.value || '');
  }, []);

  const handleFilterChange = useCallback((filters: TaskFilters<CommandType>): void => {
    storage.set('filters', filters);
    setFilters(filters);
  }, [ setFilters, storage ]);

  const handleTableRow = useCallback((record: CommandTask) => ({
    onClick: canBeOpened(record) ? makeClickHandler(record.url as string) : undefined,
  }), []);

  // TODO select and batch operation:
  // https://ant.design/components/table/#components-table-demo-row-selection-and-operation
  return (
    <Page title="Tasks">
      <div className={css.base}>
        <div className={css.header}>
          <Input
            allowClear
            className={css.search}
            placeholder="ID or name"
            prefix={<Icon name="search" size="small" />}
            onChange={handleSearchChange} />
          <TaskFilter<CommandType>
            filters={filters}
            showExperiments={false}
            showLimit={false}
            onChange={handleFilterChange} />
        </div>
        <Table
          columns={columns}
          dataSource={filteredTasks}
          loading={!hasLoaded}
          rowClassName={(record): string => canBeOpened(record) ? linkCss.base : ''}
          rowKey="id"
          size="small"
          onRow={handleTableRow} />
      </div>
    </Page>
  );
};

export default TaskList;
