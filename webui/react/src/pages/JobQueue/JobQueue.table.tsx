import { Tooltip } from 'antd';
import { ColumnType } from 'antd/es/table';
import React, { ReactNode } from 'react';

import Avatar from 'components/Avatar';
import Badge, { BadgeType } from 'components/Badge';
import Icon from 'components/Icon';
import Link from 'components/Link';
import { relativeTimeRenderer } from 'components/Table';
import { paths } from 'routes/utils';
import { getJupyterLabs, getTensorBoards } from 'services/api';
import { Job, JobType } from 'types';
import { jobTypeIconName, jobTypeLabel } from 'utils/job';
import { floatToPercent, truncate } from 'utils/string';
import { openCommand } from 'wait';

import css from './JobQueue.module.scss';

type Renderer<T> = (_: unknown, record: T) => ReactNode;
export type JobTypeRenderer = Renderer<Job>;

export const SCHEDULING_VAL_KEY = 'schedulingVal';

const routeToTask = async (taskId: string, jobType: JobType): Promise<void> => {
  let cmds = [];
  switch (jobType) {
    case JobType.TENSORBOARD:
      cmds = await getTensorBoards({});
      break;
    case JobType.NOTEBOOK:
      cmds = await getJupyterLabs({});
      break;
    default:
      throw new Error(`Unsupported job type: ${jobType}`);
  }

  const task = cmds.find(t => t.id === taskId);
  if (task) {
    openCommand(task);
  } else {
    throw new Error(`${jobType} ${taskId} not found`);
  }
};

const linkToEntityPage = (job: Job, label: ReactNode): ReactNode => {
  switch (job.type) {
    case JobType.EXPERIMENT:
      return <Link path={paths.experimentDetails(job.entityId)}>{label}</Link>;
    case JobType.NOTEBOOK:
    case JobType.TENSORBOARD:
      return (
        <Link onClick={() => {
          routeToTask(job.entityId, job.type);
        }}>{label}
        </Link>
      );
    default:
      return label;
  }
};

export const columns: ColumnType<Job>[] = [
  { key: 'jobsAhead' },
  // { // We might want to show the entityId here instead.
  //   dataIndex: 'jobId',
  //   key: 'jobId',
  //   render: (_: unknown, record: Job): ReactNode => {
  //     const label = truncate(record.jobId, 6, '');
  //     return linkToEntityPage(record, label);
  //   },
  //   title: 'ID',
  // },
  {
    dataIndex: 'type',
    key: 'type',
    render: (_: unknown, record: Job): ReactNode => {
      const title = jobTypeLabel(record.type);
      const TypeCell = (
        <Tooltip placement="topLeft" title={title}>
          <div>
            <Icon name={jobTypeIconName(record.type)} />
          </div>
        </Tooltip>
      );
      return TypeCell;
    },
    title: 'Type',
  },
  {
    key: 'name',
    render: (_: unknown, record: Job): ReactNode => {
      let label: ReactNode = null;
      switch (record.type) {
        case JobType.EXPERIMENT:
          label = (
            <div>{record.name}
              <Tooltip title="Experiment ID">
                {` (${record.entityId})`}
              </Tooltip>
            </div>
          );
          break;
        default:
          label = <span>{jobTypeLabel(record.type)} {truncate(record.entityId, 6, '')}</span>;
          break;
      }

      return linkToEntityPage(record, label);
    },
    title: 'Job Name',
  },
  {
    dataIndex: 'priority',
    key: SCHEDULING_VAL_KEY,
    title: 'Priority',
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
      const cell = (
        <span>
          <Tooltip title="Allocated (scheduled) slots">{record.allocatedSlots}</Tooltip>
          {' / '}
          <Tooltip title="Requested (queued) slots">{record.requestedSlots}</Tooltip>
        </span>
      );
      return cell;
    },
    title: 'Slots',
  },
  {
    key: 'state',
    render: (_: unknown, record: Job): ReactNode => {
      return (
        <div className={css.state}>
          <Badge state={record.summary.state} type={BadgeType.State} />
          {(!!record?.progress) && <span> {floatToPercent(record.progress, 1)}</span>}
        </div>
      );
    },
    title: 'Status',
  },
  {
    key: 'user',
    render: (_: unknown, record: Job): ReactNode => {
      const cell = <Avatar userId={record.userId} />;
      return cell;
    },
    title: 'User',
  },
  {
    align: 'right',
    className: 'fullCell',
    fixed: 'right',
    key: 'actions',
    title: '',
    width: 40,
  },
];
