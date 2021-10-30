import { Tooltip } from 'antd';
import { ColumnType } from 'antd/es/table';
import React, { ReactNode } from 'react';

import Avatar from 'components/Avatar';
import Badge, { BadgeType } from 'components/Badge';
import Icon from 'components/Icon';
import Link from 'components/Link';
import { relativeTimeRenderer } from 'components/Table';
import { paths } from 'routes/utils';
import { Job, JobType } from 'types';
import { jobTypeIconName, jobTypeLabel } from 'utils/job';
import { floatToPercent, truncate } from 'utils/string';

type Renderer<T> = (_: unknown, record: T) => ReactNode;
export type JobTypeRenderer = Renderer<Job>;

export const SCHEDULING_VAL_KEY = 'schedulingVal';

// translate job type to paths from routes utils
const jobTypeToPath = (type: JobType, id: string): string => {
  switch (type) {
    case JobType.EXPERIMENT:
      return paths.experimentDetails(id);
    case JobType.NOTEBOOK:
    case JobType.TENSORBOARD:
      // FIXME we need the command's service address
      return paths.taskList();
    default:
      return '';
  }
};

export const columns: ColumnType<Job>[] = [
  { key: 'jobsAhead' },
  {
    dataIndex: 'jobId',
    key: 'jobId',
    render: (_: unknown, record: Job): ReactNode => {
      const pagePath = jobTypeToPath(record.type, record.entityId);
      const label = truncate(record.jobId, 6, '');
      if (pagePath) {
        return <Link path={pagePath}>{label}</Link>;
      }
      return label;
    },
    title: 'ID',
  },
  {
    dataIndex: 'type',
    key: 'type',
    render: (_: unknown, record: Job): ReactNode => {
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
    render: (_: unknown, record: Job): ReactNode => {
      const pagePath = jobTypeToPath(record.type, record.entityId);
      const label = record.name ?
        record.name : <span>{jobTypeLabel(record.type)} {truncate(record.entityId, 6, '')}</span>;
      if (pagePath) {
        return <Link path={pagePath}>{label}</Link>;
      }
      return label;
    },
    title: 'Job Name',
  },
  {
    dataIndex: 'priority',
    key: SCHEDULING_VAL_KEY,
    title: 'Priority', // TODO or weight? in fairshare
  },
  {
    dataIndex: 'submissionTime',
    key: 'submitted',
    render: (_: unknown, record: Job): ReactNode =>
      record.submissionTime && relativeTimeRenderer(record.submissionTime),
    title: 'Submitted',
  },
  {
    key: 'slots',
    render: (_: unknown, record: Job): ReactNode => {
      const cell = <span>
        <Tooltip title="Allocated (scheduled) slots">{record.allocatedSlots}</Tooltip>
        {' / '}
        <Tooltip title="Requested (queued) slots">{record.requestedSlots}</Tooltip>
      </span>;
      return cell;
    },
    title: 'Slots',
  },
  {
    key: 'state',
    render: (_: unknown, record: Job): ReactNode => {
      return <>
        <Badge state={record.summary.state} type={BadgeType.State} />
        {(!!record?.progress) && <span> {floatToPercent(record.progress, 1)}</span>}
      </>;
    },
    title: 'Status',
  },
  {
    dataIndex: 'user',
    key: 'user',
    render: (_: unknown, record: Job): ReactNode => {
      const cell = <Avatar name={record.username || 'Unavailable'} />; // FIXME
      return cell;
    },
    title: 'User',
  },
];

if (process.env.IS_DEV) {
  columns.push(
    {
      align: 'right',
      className: 'fullCell',
      fixed: 'right',
      key: 'actions',
      title: '',
      width: 40,
    },
  );
}
