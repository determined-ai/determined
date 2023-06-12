import React from 'react';

import Icon from 'components/kit/Icon';
import { ansiToHtml, capitalize } from 'components/kit/internal/functions';
import { LogLevel } from 'components/kit/internal/types';
import Tooltip from 'components/kit/Tooltip';

import css from './LogViewerEntry.module.scss';

export interface LogEntry {
  formattedTime: string;
  level: LogLevel;
  message: string;
}

export interface Props extends LogEntry {
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

const LogViewerEntry: React.FC<Props> = ({
  level = LogLevel.None,
  message,
  noWrap = false,
  style,
  formattedTime,
  timeStyle,
}) => {
  const classes = [css.base];
  const levelClasses = [css.level, css[level]];
  const messageClasses = [css.message, css[level]];

  if (noWrap) classes.push(css.noWrap);

  return (
    <div className={classes.join(' ')} style={style} tabIndex={0}>
      <Tooltip content={`Level: ${capitalize(level)}`} placement="top">
        <div className={levelClasses.join(' ')} style={{ width: ICON_WIDTH }}>
          <div className={css.levelLabel}>&lt;[{level}]&gt;</div>
          {level !== LogLevel.None && <Icon name={level} size="small" title={level} />}
        </div>
      </Tooltip>
      <div className={css.time} style={timeStyle}>
        {formattedTime}
      </div>
      <div
        className={messageClasses.join(' ')}
        dangerouslySetInnerHTML={{ __html: ansiToHtml(message) }}
      />
    </div>
  );
};

export default LogViewerEntry;
