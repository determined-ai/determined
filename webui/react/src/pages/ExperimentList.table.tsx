import { ColumnType } from 'antd/es/table';
import React from 'react';

import {
  experimentArchivedRenderer, experimentProgressRenderer,
  expermentDurationRenderer, relativeTimeRenderer, stateRenderer, userRenderer,
} from 'components/Table';
import { V1GetExperimentsRequestSortBy } from 'services/api-ts-sdk';
import { ExperimentItem } from 'types';

export const columns: ColumnType<ExperimentItem>[] = [
  {
    dataIndex: 'id',
    key: V1GetExperimentsRequestSortBy.ID,
    sorter: true,
    title: 'ID',
  },
  {
    dataIndex: 'name',
    key: V1GetExperimentsRequestSortBy.DESCRIPTION,
    sorter: true,
    title: 'Name',
  },
  {
    key: V1GetExperimentsRequestSortBy.STARTTIME,
    render: (_: number, record: ExperimentItem): React.ReactNode =>
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
    key: V1GetExperimentsRequestSortBy.NUMTRIALS,
    sorter: true,
    title: 'Trials',
  },
  {
    key: V1GetExperimentsRequestSortBy.STATE,
    render: stateRenderer,
    sorter: true,
    title: 'State',
  },
  {
    key: V1GetExperimentsRequestSortBy.PROGRESS,
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
    key: V1GetExperimentsRequestSortBy.USER,
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
