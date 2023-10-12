import React, { ReactNode } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import Icon from 'components/kit/Icon';
import Tooltip from 'components/kit/Tooltip';
import Link from 'components/Link';
import { ColumnDef } from 'components/Table/InteractiveTable';
import { createOmitableRenderer, relativeTimeRenderer } from 'components/Table/Table';
import { paths } from 'routes/utils';
import { getJupyterLabs, getTensorBoards } from 'services/api';
import { CommandTask, FullJob, Job, JobType } from 'types';
import { jobTypeIconName, jobTypeLabel } from 'utils/job';
import { floatToPercent, truncate } from 'utils/string';
import { openCommand } from 'utils/wait';

import css from './JobQueue.module.scss';
import { DEFAULT_COLUMN_WIDTHS } from './JobQueue.settings';

type Renderer<T> = (_: unknown, record: T) => ReactNode;
export type JobTypeRenderer = Renderer<Job>;

export const SCHEDULING_VAL_KEY = 'schedulingVal';

const routeToTask = async (taskId: string, jobType: JobType): Promise<void> => {
  let cmds: CommandTask[] = [];
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

  const task = cmds.find((t) => t.id === taskId);
  if (task) {
    openCommand(task);
  } else {
    throw new Error(`${jobType} ${taskId} not found`);
  }
};

const linkToEntityPage = (job: Job, label: ReactNode): ReactNode => {
  if (!('entityId' in job)) return label;
  switch (job.type) {
    case JobType.EXPERIMENT:
      return <Link path={paths.experimentDetails(job.entityId)}>{label}</Link>;
    case JobType.NOTEBOOK:
    case JobType.TENSORBOARD:
      return (
        <Link
          onClick={() => {
            routeToTask(job.entityId, job.type);
          }}>
          {label}
        </Link>
      );
    default:
      return label;
  }
};

export const columns: ColumnDef<Job>[] = [
  {
    align: 'center',
    dataIndex: 'preemptible',
    defaultWidth: DEFAULT_COLUMN_WIDTHS['preemptible'],
    key: 'jobsAhead',
  },
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
    align: 'center',
    dataIndex: 'type',
    defaultWidth: DEFAULT_COLUMN_WIDTHS['type'],
    key: 'type',
    render: (_: unknown, record: Job): ReactNode => (
      <Icon name={jobTypeIconName(record.type)} showTooltip title={jobTypeLabel(record.type)} />
    ),
    title: 'Type',
  },
  {
    dataIndex: 'name',
    defaultWidth: DEFAULT_COLUMN_WIDTHS['name'],
    key: 'name',
    render: createOmitableRenderer<Job, FullJob>('entityId', (_, record): ReactNode => {
      let label: ReactNode = null;
      switch (record.type) {
        case JobType.EXPERIMENT:
          label = (
            <div>
              {record.name}
              <Tooltip content="Experiment ID">{` (${record.entityId})`}</Tooltip>
            </div>
          );
          break;
        case JobType.EXTERNAL:
          label = <div>{record.name}</div>;
          break;
        default:
          label = (
            <span>
              {jobTypeLabel(record.type)} {truncate(record.entityId, 6, '')}
            </span>
          );
          break;
      }
      return linkToEntityPage(record, label);
    }),
    title: 'Job Name',
  },
  {
    dataIndex: 'priority',
    defaultWidth: DEFAULT_COLUMN_WIDTHS['priority'],
    key: SCHEDULING_VAL_KEY,
    title: 'Priority',
  },
  {
    align: 'right',
    dataIndex: 'submissionTime',
    defaultWidth: DEFAULT_COLUMN_WIDTHS['submissionTime'],
    key: 'submitted',
    render: createOmitableRenderer<Job, FullJob>(
      'entityId',
      (_, record): ReactNode =>
        record.submissionTime && relativeTimeRenderer(record.submissionTime),
    ),
    title: 'Submitted',
  },
  {
    align: 'right',
    dataIndex: 'slots',
    defaultWidth: DEFAULT_COLUMN_WIDTHS['slots'],
    key: 'slots',
    render: (_: unknown, record: Job): ReactNode => {
      const cell = (
        <span>
          <Tooltip content="Allocated (scheduled) slots">{record.allocatedSlots}</Tooltip>
          {' / '}
          <Tooltip content="Requested (queued) slots">{record.requestedSlots}</Tooltip>
        </span>
      );
      return cell;
    },
    title: 'Slots',
  },
  {
    align: 'center',
    dataIndex: 'status',
    defaultWidth: DEFAULT_COLUMN_WIDTHS['status'],
    key: 'state',
    render: (_: unknown, record: Job): ReactNode => {
      return (
        <div className={css.state}>
          <Badge state={record.summary.state} type={BadgeType.State} />
          {!!record?.progress && <span> {floatToPercent(record.progress, 1)}</span>}
        </div>
      );
    },
    title: 'State',
  },
  {
    align: 'center',
    dataIndex: 'user',
    defaultWidth: DEFAULT_COLUMN_WIDTHS['user'],
    key: 'user',
    title: 'User',
  },
  {
    align: 'right',
    className: 'fullCell',
    dataIndex: 'action',
    defaultWidth: DEFAULT_COLUMN_WIDTHS['action'],
    fixed: 'right',
    key: 'actions',
    title: '',
  },
];
