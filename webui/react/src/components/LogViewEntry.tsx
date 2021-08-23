import React from 'react';

import { LogLevel } from 'types';
import { ansiToHtml } from 'utils/dom';

import css from './LogViewerEntry.module.scss';
import LogViewerLevel from './LogViewerLevel';

interface Prop {
  formattedTime: string;
  level: LogLevel;
  message: string;
  style?: React.CSSProperties;
  timeStyle?: React.CSSProperties;
}

const LogViewerEntry: React.FC<Prop> = ({ level, message, style, formattedTime, timeStyle }) => {
  const messageClasses = [ css.message ];

  if (level) messageClasses.push(css[level]);

  return (
    <div className={css.base} style={style}>
      <LogViewerLevel logLevel={level} />
      <div className={css.time} style={timeStyle}>{formattedTime}</div>
      <div
        className={messageClasses.join(' ')}
        dangerouslySetInnerHTML={{ __html: ansiToHtml(message) }}
      />
    </div>
  );
};

export default LogViewerEntry;
