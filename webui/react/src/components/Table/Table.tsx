import { Space, Tooltip, Typography } from 'antd';
import React from 'react';

import Badge, { BadgeType } from 'components/Badge';
import { ConditionalWrapper } from 'components/ConditionalWrapper';
import ExperimentIcons from 'components/ExperimentIcons';
import HumanReadableNumber from 'components/HumanReadableNumber';
import Link from 'components/Link';
import ProgressBar from 'components/ProgressBar';
import TimeAgo from 'components/TimeAgo';
import TimeDuration from 'components/TimeDuration';
import UserAvatar from 'components/UserAvatar';
import { commandTypeToLabel } from 'constants/states';
import { paths } from 'routes/utils';
import Icon from 'shared/components/Icon/Icon';
import Spinner from 'shared/components/Spinner';
import { Pagination } from 'shared/types';
import { getDuration } from 'shared/utils/datetime';
import { useUsers } from 'stores/users';
import { StateOfUnion } from 'themes';
import {
  CommandTask,
  CommandType,
  DetailedUser,
  ExperimentItem,
  ModelItem,
  ModelVersion,
  Project,
  RunState,
  StartEndTimes,
  TrialItem,
  Workspace,
} from 'types';
import { Loadable } from 'utils/loadable';
import { canBeOpened } from 'utils/task';
import { openCommand } from 'utils/wait';

import css from './Table.module.scss';

type TableRecord = CommandTask | ExperimentItem | TrialItem | Project | Workspace;

export interface TablePaginationConfig {
  current: number;
  defaultPageSize: number;
  hideOnSinglePage: boolean;
  pageSize: number;
  showSizeChanger: boolean;
  total: number;
}

export type Renderer<T = unknown> = (text: string, record: T, index: number) => React.ReactNode;

export type GenericRenderer<T extends TableRecord> = (
  text: string,
  record: T,
  index: number,
) => React.ReactNode;

export type ExperimentRenderer = (
  text: string,
  record: ExperimentItem,
  index: number,
) => React.ReactNode;

export type TaskRenderer = (text: string, record: CommandTask, index: number) => React.ReactNode;

export const MINIMUM_PAGE_SIZE = 10;

export const defaultPaginationConfig = {
  current: 1,
  defaultPageSize: MINIMUM_PAGE_SIZE,
  pageSize: MINIMUM_PAGE_SIZE,
  showSizeChanger: true,
};

/* Table Column Renderers */

export const checkmarkRenderer = (yesNo: boolean): React.ReactNode => {
  return yesNo ? <Icon name="checkmark" /> : null;
};

export const durationRenderer = (times: StartEndTimes): React.ReactNode => (
  <TimeDuration duration={getDuration(times)} />
);

export const HumanReadableNumberRenderer = (num: number): React.ReactNode => {
  return <HumanReadableNumber num={num} />;
};

export const relativeTimeRenderer = (date: Date): React.ReactNode => {
  return <TimeAgo className={css.timeAgo} datetime={date} />;
};

export const stateRenderer: Renderer<{ state: StateOfUnion }> = (_, record) => (
  <div className={`${css.centerVertically} ${css.centerHorizontally}`}>
    <Badge state={record.state} type={BadgeType.State} />
  </div>
);

export const expStateRenderer: Renderer<{ state: RunState }> = (_, record) => (
  <div className={`${css.centerVertically} ${css.centerHorizontally}`}>
    <ExperimentIcons state={record.state} />
  </div>
);

export const tooltipRenderer: Renderer = (text) => (
  <Tooltip placement="topLeft" title={text}>
    <span>{text}</span>
  </Tooltip>
);

export const userRenderer: React.FC<DetailedUser | undefined> = (user) => {
  return (
    <div className={`${css.centerVertically} ${css.centerHorizontally}`}>
      {user ? <UserAvatar user={user} /> : <Spinner />}
    </div>
  );
};

export const UserRenderer: Renderer<{ userId?: number }> = (_, record) => {
  const users = Loadable.getOrElse([], useUsers());
  const user = users.find((user) => user.id === record.userId);
  return (
    <div className={`${css.centerVertically} ${css.centerHorizontally}`}>
      {user ? <UserAvatar user={user} /> : <Spinner />}
    </div>
  );
};

/* Command Task Table Column Renderers */

export const taskIdRenderer: TaskRenderer = (_, record) => (
  <Tooltip placement="topLeft" title={record.id}>
    <div className={css.centerVertically}>
      <ConditionalWrapper
        condition={canBeOpened(record)}
        wrapper={(children) => <Link onClick={() => openCommand(record)}>{children}</Link>}>
        <Badge type={BadgeType.Id}>{record.id.split('-')[0]}</Badge>
      </ConditionalWrapper>
    </div>
  </Tooltip>
);

export const taskTypeRenderer: TaskRenderer = (_, record) => (
  <Tooltip placement="topLeft" title={commandTypeToLabel[record.type as unknown as CommandType]}>
    <div className={css.centerVertically}>
      <Icon name={record.type.toLowerCase()} />
    </div>
  </Tooltip>
);

export const taskNameRenderer: TaskRenderer = (id, record) => (
  <div>
    <ConditionalWrapper
      condition={canBeOpened(record)}
      wrapper={(ch) => (
        <a href={`${process.env.PUBLIC_URL}${paths.interactive(record)}`} target={record.id}>
          {ch}
        </a>
      )}>
      <span>{record.name}</span>
    </ConditionalWrapper>
  </div>
);

/* Experiment Table Column Renderers */

export const experimentDurationRenderer: ExperimentRenderer = (_, record) => (
  <TimeDuration duration={getDuration(record)} />
);

export const experimentNameRenderer = (
  value: string | number | undefined,
  record: ExperimentItem,
): React.ReactNode => (
  <Typography.Text ellipsis={{ tooltip: true }}>
    <Link path={paths.experimentDetails(record.id)}>{value === undefined ? '' : value}</Link>
  </Typography.Text>
);

export const experimentProgressRenderer: ExperimentRenderer = (_, record) => {
  return typeof record.progress !== 'undefined' ? (
    <ProgressBar percent={record.progress * 100} state={record.state} />
  ) : null;
};

/* Model Table Column Renderers */

export const modelNameRenderer = (value: string, record: ModelItem): React.ReactNode => (
  <Space className={css.wordBreak}>
    <div style={{ paddingInline: 4 }}>
      <Icon name="model" size="medium" />
    </div>
    <Link path={paths.modelDetails(String(record.id))}>{value}</Link>
  </Space>
);

export const modelVersionNameRenderer = (value: string, record: ModelVersion): React.ReactNode => (
  <Link path={paths.modelVersionDetails(String(record.model.id), record.version)}>
    {value ? value : 'Version ' + record.version}
  </Link>
);

export const modelVersionNumberRenderer = (
  value: string,
  record: ModelVersion,
): React.ReactNode => (
  <Link
    className={css.versionBox}
    path={paths.modelVersionDetails(String(record.model.id), record.version)}>
    V{record.version}
  </Link>
);

/* Table Helper Functions */

/*
 * For an `onClick` event on a table row, sometimes we have alternative and secondary
 * click interactions we want to capture. For example, we might want to capture different
 * link besides the one the table row is linked to. This function provides the means to
 * detect these alternative actions based on className definitions.
 */
export const isAlternativeAction = (event: React.MouseEvent): boolean => {
  const target = event.target as Element;
  if (
    target.className.includes('ant-checkbox-wrapper') ||
    target.className.includes('ignoreTableRowClick')
  )
    return true;
  return false;
};

/*
 * Default clickable row class name for Table components.
 */
export const defaultRowClassName = (options?: {
  clickable?: boolean;
  highlighted?: boolean;
}): string => {
  const classes: string[] = [];
  if (options?.clickable) classes.push('clickable');
  if (options?.highlighted) classes.push('highlighted');
  return classes.join(' ');
};

export const getPaginationConfig = (
  count: number,
  pageSize?: number,
): Partial<TablePaginationConfig> => {
  return {
    defaultPageSize: MINIMUM_PAGE_SIZE,
    hideOnSinglePage: count < MINIMUM_PAGE_SIZE,
    pageSize,
    showSizeChanger: true,
  };
};

export const getFullPaginationConfig = (
  pagination: Pagination,
  total: number,
): TablePaginationConfig => {
  return {
    current: Math.floor(pagination.offset / pagination.limit) + 1,
    defaultPageSize: MINIMUM_PAGE_SIZE,
    hideOnSinglePage: total < MINIMUM_PAGE_SIZE,
    pageSize: pagination.limit,
    showSizeChanger: true,
    total,
  };
};
