import { Tooltip } from 'antd';
import React from 'react';

import { LogLevel } from '../types';
import { toRem } from '../utils/dom';
import { capitalize } from '../utils/string';

import Icon from './Icon';
import css from './LogViewer.module.scss';

export const ICON_WIDTH = 26;

interface Props {
  logLevel?: LogLevel;
}

const LogViewerLevel: React.FC<Props> = ({ logLevel }) => {
  const levelStyle = { width: toRem(ICON_WIDTH) };

  if (!logLevel) {
    logLevel = LogLevel.Info;
  }

  const classes = [ css.level, logLevel ];

  return (
    <Tooltip placement="top" title={`Level: ${capitalize(logLevel)}`}>
      <div className={classes.join(' ')} style={levelStyle}>
        <div className={css.levelLabel}>&lt;[{logLevel}]&gt;</div>
        <Icon name={logLevel} size="small" />
      </div>
    </Tooltip>
  );
};

export default LogViewerLevel;
