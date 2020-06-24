import { Table } from 'antd';
import { ColumnsType, ColumnType } from 'antd/lib/table';
import React, { MouseEventHandler } from 'react';

import Badge from 'components/Badge';
import linkCss from 'components/Link.module.scss';
import { actionsColumn, Renderer, startTimeColumn, stateColumn,
  userColumn } from 'table/columns';
import { CommonProps, Experiment } from 'types';
import { alphanumericSorter } from 'utils/data';
import { floatToPercent } from 'utils/string';
import { experimentDuration, shortEnglishHumannizer } from 'utils/time';
import { experimentToTask } from 'utils/types';

import css from './ExperimentTable.module.scss';
import ProgressBar from './ProgressBar';
import { tableRowClickHandler } from './TaskTable';

interface Props extends CommonProps {
  experiments?: Experiment[];
}

const progressRenderer: Renderer<Experiment> = (_, record) => {
  if (!record.progress) return;
  return <ProgressBar
    percent={record.progress * 100}
    state={record.state}
    title={floatToPercent(record.progress, 0)} />;
};

const durationRenderer: Renderer<Experiment> = (_, record) => {
  return shortEnglishHumannizer(experimentDuration(record));
};

const descriptionRenderer: Renderer<Experiment> = (_, record) => {
  // TODO handle displaying labels not fitting the column width
  const labels = [ 'object detection', 'pytorch' ]; // TODO get from config
  const labelEls = labels.map((text, idx) => <Badge key={idx}>{text}</Badge>);
  return (
    <div className={css.nameColumn}>
      <div>{record.config.description}</div>
      <div>{labelEls}</div>
    </div>
  );
};

const columns: ColumnsType<Experiment> = [
  {
    dataIndex: 'id',
    sorter: (a, b): number => alphanumericSorter(a.id, b.id),
    title: 'ID',
  },
  {
    render: descriptionRenderer,
    sorter: (a, b): number => alphanumericSorter(a.config.description, b.config.description),
    title: 'Name',
  },
  startTimeColumn as ColumnType<Experiment>,
  {
    render: durationRenderer,
    title: 'Duration',
  },
  {
    // TODO bring in actual trial counts once available.
    render: (): number => Math.floor(Math.random() * 100),
    title: 'Trials',
  },
  stateColumn as ColumnType<Experiment>,
  {
    render: progressRenderer,
    title: 'Progress',
  },
  userColumn as ColumnType<Experiment>,
  actionsColumn as ColumnType<Experiment>,
];

const ExperimentsTable: React.FC<Props> = ({ experiments }: Props) => {
  return (
    <Table
      className={css.base}
      columns={columns}
      dataSource={experiments}
      loading={experiments === undefined}
      rowClassName={(): string => linkCss.base}
      rowKey="id"
      onRow={(record: Experiment): {onClick?: MouseEventHandler} =>
        tableRowClickHandler(experimentToTask(record))} />

  );
};

export default ExperimentsTable;
