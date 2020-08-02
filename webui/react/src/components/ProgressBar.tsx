import { Tooltip } from 'antd';
import React from 'react';

import { getStateColor } from 'themes';
import { CommandState, RunState } from 'types';
import { floatToPercent } from 'utils/string';

import css from './ProgressBar.module.scss';

interface Props {
  percent: number;
  state: RunState | CommandState;
}

const defaultProps = {
  percent: 0,
};

const ProgressBar: React.FC<Props> = ({ percent, state }: Props) => {
  const style = {
    backgroundColor: getStateColor(state),
    width: `${percent}%`,
  };

  return (
    <Tooltip title={floatToPercent(percent / 100, 0)}>
      <div className={css.base}>
        <div className={css.bar}>
          <span className={css.progress} style={style} />
        </div>
      </div>
    </Tooltip>
  );
};

ProgressBar.defaultProps = defaultProps;

export default ProgressBar;
