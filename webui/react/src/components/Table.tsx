import { Tooltip } from 'antd';
import React from 'react';
import TimeAgo from 'timeago-react';

import Avatar from 'components/Avatar';
import Badge, { BadgeType } from 'components/Badge';
import HumanReadableFloat from 'components/HumanReadableFloat';
import Icon from 'components/Icon';
import ProgressBar from 'components/ProgressBar';
import {
  CheckpointState,
  CommandState, CommandTask, CommandType, ExperimentItem,
  Pagination, RunState, StartEndTimes, TrialItem,
} from 'types';
import { getDuration, shortEnglishHumannizer } from 'utils/time';
import { commandTypeToLabel } from 'utils/types';

import css from './Table.module.scss';

type TableRecord = CommandTask | ExperimentItem | TrialItem;

export interface TablePaginationConfig {
  current: number;
  pageSize: number;
  defaultPageSize: number;
  hideOnSinglePage: boolean;
  showSizeChanger: boolean;
  total: number;
}

export type Renderer<T = unknown> = (text: string, record: T, index: number) => React.ReactNode;

export type GenericRenderer = <T extends TableRecord>(
  text: string, record: T, index: number,
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

export const durationRenderer = (times: StartEndTimes): React.ReactNode => {
  return shortEnglishHumannizer(getDuration(times));
};

export const humanReadableFloatRenderer = (num: number): React.ReactNode => {
  return <HumanReadableFloat num={num} />;
};

export const relativeTimeRenderer = (date: Date): React.ReactNode => {
  return (
    <Tooltip title={date.toLocaleString()}>
      <TimeAgo datetime={date} />
    </Tooltip>
  );
};

export const stateRenderer: Renderer<{ state: CommandState | RunState | CheckpointState }> =
(_, record) => (
  <div className={css.centerVertically}>
    <Badge state={record.state} type={BadgeType.State} />
  </div>
);

export const tooltipRenderer: Renderer = text => (
  <Tooltip placement="topLeft" title={text}><span>{text}</span></Tooltip>
);

export const userRenderer: Renderer<{ username: string }> = (_, record) => (
  <Avatar name={record.username} />
);

/* Command Task Table Column Renderers */

export const taskIdRenderer: TaskRenderer = id => (
  <Tooltip placement="topLeft" title={id}>
    <div className={css.centerVertically}>
      <Badge type={BadgeType.Id}>{id.split('-')[0]}</Badge>
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

/* Experiment Table Column Renderers */

export const experimentDescriptionRenderer: ExperimentRenderer = (_, record) => {
  // TODO handle displaying labels not fitting the column width
  const labels = [ 'object detection', 'pytorch' ]; // TODO get from config
  const labelEls = labels.map((text, idx) => <Badge key={idx}>{text}</Badge>);
  return (
    <div className={css.nameColumn}>
      <div>{record.name || ''}</div>
      <div>{labelEls}</div>
    </div>
  );
};

export const expermentDurationRenderer: ExperimentRenderer = (_, record) => {
  return shortEnglishHumannizer(getDuration(record));
};

export const experimentProgressRenderer: ExperimentRenderer = (_, record) => {
  return record.progress ? <ProgressBar
    percent={record.progress * 100}
    state={record.state} /> : null;
};

export const experimentArchivedRenderer: ExperimentRenderer = (_, record) => {
  return record.archived ? <Icon name="checkmark" /> : null;
};

/* Table Helper Functions */

/*
 * For an `onClick` event on a table row, sometimes we have alternative and secondary
 * click interactions we want to capture. For example, we might want to capture different
 * link besides the one the table row is linked to. This function provides the means to
 * detect these alternative actions based on className definitions.
 */
export const isAlternativeAction = (event: React.MouseEvent): boolean => {
  const target = event.target as Element;
  if (target.className.includes('ant-checkbox-wrapper') ||
      target.className.includes('ignoreTableRowClick')) return true;
  return false;
};

/*
 * Default clickable row class name for Table components.
 */
export const defaultRowClassName = (clickable = true): string=> {
  return clickable ? 'clickable' : '';
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
