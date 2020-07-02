import { Table } from 'antd';
import React from 'react';

import { columns as experimentColumns } from 'pages/ExperimentList.table';
import { columns as taskColumns } from 'pages/TaskList.table';
import { ExperimentsDecorator } from 'storybook/ConetextDecorators';
import RouterDecorator from 'storybook/RouterDecorator';
import { CommandTask, ExperimentItem } from 'types';
import { generateCommandTask, generateExperiments } from 'utils/task';

import linkCss from './Link.module.scss';
import css from './Table.module.scss';

export default {
  component: Table,
  decorators: [ RouterDecorator, ExperimentsDecorator ],
  title: 'Table',
};

const commandTasks: CommandTask[] = new Array(20)
  .fill(null)
  .map((_, index) => generateCommandTask(index));
const experimentItems: ExperimentItem[] = generateExperiments(30);

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
    rowClassName={(): string => linkCss.base}
    rowKey="id" />;
};

export const ExperimentTable = (): React.ReactNode => {
  return <Table
    className={css.base}
    columns={experimentColumns}
    dataSource={experimentItems}
    loading={experimentItems === undefined}
    rowClassName={(): string => linkCss.base}
    rowKey="id" />;
};
