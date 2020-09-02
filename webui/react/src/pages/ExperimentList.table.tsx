import { ColumnType } from 'antd/lib/table';
import React from 'react';

import {
  experimentArchivedRenderer, experimentProgressRenderer,
  expermentDurationRenderer, relativeTimeRenderer, stateRenderer, userRenderer,
} from 'components/Table';
import { ExperimentX } from 'types';

export const columns: ColumnType<ExperimentX>[] = [
  {
    dataIndex: 'id',
    key: 'id',
    sorter: true,
    title: 'ID',
  },
  {
    dataIndex: 'name',
    key: 'name',
    sorter: true,
    title: 'Name',
  },
  {
    key: 'startTime',
    render: (_: number, record: ExperimentX): React.ReactNode =>
      relativeTimeRenderer(new Date(record.startTime)),
    sorter: true,
    title: 'Start Time',
  },
  {
    key: 'duration',
    render: expermentDurationRenderer,
    title: 'Duration',
  },
  {
    dataIndex: 'numTrials',
    key: 'numTrials',
    sorter: true,
    title: 'Trials',
  },
  {
    key: 'state',
    render: stateRenderer,
    sorter: true,
    title: 'State',
  },
  {
    key: 'progress',
    render: experimentProgressRenderer,
    sorter: true,
    title: 'Progress',
  },
  {
    key: 'archived',
    render: experimentArchivedRenderer,
    title: 'Archived',
  },
  {
    key: 'user',
    render: userRenderer,
    sorter: true,
    title: 'User',
  },
  {
    align: 'right',
    className: 'fullCell',
    key: 'action',
    title: '',
  },
];
