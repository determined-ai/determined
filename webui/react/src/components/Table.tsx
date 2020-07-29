import { CheckOutlined } from '@ant-design/icons';
import { Tooltip } from 'antd';
import React from 'react';
import TimeAgo from 'timeago-react';

import Avatar from 'components/Avatar';
import Badge, { BadgeType } from 'components/Badge';
import Icon from 'components/Icon';
import ProgressBar from 'components/ProgressBar';
import TaskActionDropdown from 'components/TaskActionDropdown';
import {
  CommandState, CommandTask, CommandType, ExperimentItem, RunState, StartEndTimes, TrialItem,
} from 'types';
import { floatToPercent } from 'utils/string';
import { getDuration, shortEnglishHumannizer } from 'utils/time';
import { commandTypeToLabel, experimentToTask } from 'utils/types';

import css from './Table.module.scss';

type TableRecord = CommandTask | ExperimentItem | TrialItem;

export type Renderer<T = unknown> = (text: string, record: T, index: number) => React.ReactNode;

export type GenericRenderer = <T extends TableRecord>(
  text: string, record: T, index: number,
) => React.ReactNode;

type ExperimentRenderer = (text: string, record: ExperimentItem, index: number) => React.ReactNode;
type TaskRenderer = (text: string, record: CommandTask, index: number) => React.ReactNode;

/* Table Column Renderers */

export const durationRenderer = (times: StartEndTimes): React.ReactNode => {
  return shortEnglishHumannizer(getDuration(times));
};

export const relativeTimeRenderer = (date: Date): React.ReactNode => {
  return (
    <Tooltip title={date.toLocaleString()}>
      <TimeAgo datetime={date} />
    </Tooltip>
  );
};

export const stateRenderer: Renderer<{ state: CommandState | RunState }> = (_, record) => (
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

export const taskActionRenderer: TaskRenderer = (_, record) => <TaskActionDropdown task={record} />;

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

export const experimentActionRenderer: ExperimentRenderer = (_, record) => (
  <TaskActionDropdown task={experimentToTask(record)} />
);

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
  if (!record.progress) return;
  return <ProgressBar
    percent={record.progress * 100}
    state={record.state}
    title={floatToPercent(record.progress, 0)} />;
};

export const experimentArchivedRenderer: ExperimentRenderer = (_, record) => {
  return record.archived ? <CheckOutlined /> : null;
};
