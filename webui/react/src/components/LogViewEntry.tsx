import { Tooltip } from 'antd';
import React from 'react';

import { LogLevel } from 'types';
import { ansiToHtml } from 'utils/dom';
import { capitalize } from 'utils/string';

import Icon from './Icon';
import css from './LogViewerEntry.module.scss';

interface Prop {
  formattedTime: string;
  level: LogLevel;
  message: string;
  style?: React.CSSProperties;
  timeStyle?: React.CSSProperties;
}

export const ICON_WIDTH = 26;

const LogViewerEntry: React.FC<Prop> = ({ level, message, style, formattedTime, timeStyle }) => {
  const logLevel = level ? level : LogLevel.Info;
  const levelClasses = [ css.level, logLevel ];
  const messageClasses = [ css.message ];

  if (logLevel) messageClasses.push(css[logLevel]);

  return (
    <div className={css.base} style={style}>
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
