import { Tooltip } from 'antd';
import { ColumnType } from 'antd/es/table';
import React, { ReactNode } from 'react';

import Avatar from 'components/Avatar';
import Badge, { BadgeType } from 'components/Badge';
import Icon from 'components/Icon';
import Link from 'components/Link';
import { relativeTimeRenderer } from 'components/Table';
import { Job, ResourcePool } from 'types';
import { alphanumericSorter, numericSorter } from 'utils/sort';
import { capitalize, truncate } from 'utils/string';
import { jobTypeIconName, jobTypeLabel, V1ResourcePoolTypeToLabel } from 'utils/types';

type Renderer<T> = (_: any, record: T) => ReactNode;
export type JobTypeRenderer = Renderer<Job>;

// #, id, type, job name, pri, sumbitted, slots, progress, runtime eta, user
export const columns: ColumnType<Job>[] = [
  {
    key: 'jobsAhead',
    render: (_, record: Job): ReactNode => {
      const cell = (
        <div>
          {record.summary.jobsAhead}
          {!record.isPreemptible && <Icon name="lock" />}
        </div>
      );
      return cell;
    },
    // TODO connect to the api
    sorter: (a: Job, b: Job): number =>
      numericSorter(a.summary.jobsAhead, b.summary.jobsAhead),
    title: '#',
  },
  {
    dataIndex: 'jobId',
    key: 'jobId',
    render: (_, record: Job): ReactNode => {
      const cell = <span>
        {truncate(record.jobId, 6, '')}
      </span>;
      return cell;
    },
    title: 'ID',
  },
  {
    dataIndex: 'type',
    key: 'type',
    render: (_: string, record: Job): ReactNode => {
      const title = jobTypeLabel(record.type);
      const TypeCell = <Tooltip placement="topLeft" title={title}>
        <div className={''}>
          <Icon name={jobTypeIconName(record.type)} />
        </div>
      </Tooltip>;
      return TypeCell;
    },
    title: 'Type',
  },
  {
    key: 'name',
    render: (_, record: Job): ReactNode => {
      const cell = <Link path={'/dashboard'}>
        {jobTypeLabel(record.type)} {truncate(record.entityId, 6, '')}
      </Link>;
      return cell;
    },
    title: 'Job Name',
  },
  {
    dataIndex: 'priority',
    key: 'priority',
    // render: (_, record) => V1JobTypeToLabel[record.type],
    title: 'PRI', // TODO or weight? in fairshare
  },
  {
    dataIndex: 'submissionTime',
    key: 'submitted',
    render: (_: string, record: Job): ReactNode =>
      relativeTimeRenderer(record.submissionTime),
    title: 'Submitted',
  },
  {
    key: 'slots',
    render: (_: string, record: Job): ReactNode => {
      return <span>
        {record.allocatedSlots} / {record.requestedSlots}
      </span>;
    },
    title: 'Slots (Acquired/Requested)',
  },
  {
    key: 'state',
    render: (_, record: Job): ReactNode => {
      const cell = <Badge state={record.summary.state} type={BadgeType.State} />;
      return cell;
    },
    title: 'State',
  },
  {
    key: 'progress',
    title: 'Progress',
  },
  {
    key: 'runtime',
    render: (_, record: Job): ReactNode => {
      return 'unavailable';
    },
    title: 'Run Time',
  },
  {
    key: 'eta',
    render: (_, record: Job): ReactNode => {
      return 'unavailable';
    },
    title: 'ETA',
  },
  {
    dataIndex: 'user',
    key: 'user',
    render: (_, record: Job): ReactNode => {
      return <Avatar name={record.user} />;
    },
    title: 'User',
  },
  {
    align: 'right',
    className: 'fullCell',
    fixed: 'right',
    key: 'actions',
    title: '',
    // width: 40,
  },
];
