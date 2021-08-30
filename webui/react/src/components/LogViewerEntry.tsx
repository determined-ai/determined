import { Tooltip } from 'antd';
import React from 'react';

import { LogLevel } from 'types';
import { ansiToHtml } from 'utils/dom';
import { capitalize } from 'utils/string';

import Icon from './Icon';
import css from './LogViewerEntry.module.scss';

export interface LogEntry {
  formattedTime: string;
  level: LogLevel;
  message: string;
}

interface Prop extends LogEntry {
  noWrap?: boolean;
  style?: React.CSSProperties;
  timeStyle?: React.CSSProperties;
}

export const ICON_WIDTH = 26;

// Format the datetime to...
const DATETIME_PREFIX = '[';
const DATETIME_SUFFIX = ']';
export const DATETIME_FORMAT = `[${DATETIME_PREFIX}]YYYY-MM-DD HH:mm:ss${DATETIME_SUFFIX}`;

// Max datetime size: DATETIME_FORMAT (plus 1 for a space suffix)
export const MAX_DATETIME_LENGTH = 22;

const LogViewerEntry: React.FC<Prop> = ({
  level,
  message,
  noWrap = false,
  style,
  formattedTime,
  timeStyle,
}) => {
  const classes = [ css.base ];
  const logLevel = level ? level : LogLevel.Info;
  const levelClasses = [ css.level, logLevel ];
  const messageClasses = [ css.message ];

  if (noWrap) classes.push(css.noWrap);
  if (logLevel) messageClasses.push(css[logLevel]);

  return (
    <div className={classes.join(' ')} style={style}>
      <Tooltip placement="top" title={`Level: ${capitalize(logLevel)}`}>
        <div className={levelClasses.join(' ')} style={{ width: ICON_WIDTH }}>
          <div className={css.levelLabel}>&lt;[{logLevel}]&gt;</div>
          <Icon name={logLevel} size="small" />
        </div>
      </Tooltip>
      <div className={css.time} style={timeStyle}>{formattedTime}</div>
      <div
        className={messageClasses.join(' ')}
        dangerouslySetInnerHTML={{ __html: ansiToHtml(message) }}
      />
    </div>
  );
};

export default LogViewerEntry;
