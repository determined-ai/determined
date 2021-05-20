import { Table } from 'antd';
import React from 'react';

import { columns as taskColumns } from 'pages/TaskList.table';
import RouterDecorator from 'storybook/RouterDecorator';
import { CommandTask } from 'types';
import { generateCommandTask } from 'utils/task';

import { defaultRowClassName } from './Table';
import css from './Table.module.scss';

export default {
  component: Table,
  decorators: [ RouterDecorator ],
  parameters: { layout: 'padded' },
  title: 'Table',
};

const commandTasks: CommandTask[] = new Array(20)
  .fill(null)
  .map((_, index) => generateCommandTask(index));

export const LoadingTable = (): React.ReactNode => {
  return <Table loading={true} />;
};

export const EmptyTable = (): React.ReactNode => {
  return <Table dataSource={[]} />;
};

export const TaskTable = (): React.ReactNode => {
  return <Table
    className={css.base}
    columns={taskColumns}
    dataSource={commandTasks}
    loading={commandTasks === undefined}
    rowClassName={defaultRowClassName({ clickable: true })}
    rowKey="id"
    showSorterTooltip={false} />;
};
