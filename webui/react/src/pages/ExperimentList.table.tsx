import { ColumnType } from 'antd/es/table';
import React from 'react';

import Link from 'components/Link';
import {
  experimentArchivedRenderer, experimentProgressRenderer,
  expermentDurationRenderer, relativeTimeRenderer, stateRenderer, userRenderer,
} from 'components/Table';
import { paths } from 'routes/utils';
import { V1GetExperimentsRequestSortBy } from 'services/api-ts-sdk';
import { ExperimentItem } from 'types';

export const nameRenderer = (
  value: string | number | undefined,
  record: ExperimentItem,
): React.ReactNode => {
  return (
    <Link path={paths.experimentDetails(record.id)}>{value === undefined ? '' : value}</Link>
  );
};

export const columns: ColumnType<ExperimentItem>[] = [
  {
    dataIndex: 'id',
    key: V1GetExperimentsRequestSortBy.ID,
    render: nameRenderer,
    sorter: true,
    title: 'ID',
  },
  {
    dataIndex: 'name',
    key: V1GetExperimentsRequestSortBy.DESCRIPTION,
    render: nameRenderer,
    sorter: true,
    title: 'Name',
  },
  {
    dataIndex: 'labels',
    key: 'labels',
    title: 'Labels',
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
    dataIndex: 'resourcePool',
    key: 'resourcePool',
    sorter: true,
    title: 'Resource Pool',
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
